# EduStream — Protocole JSON (source of truth)

Ce document définit le contrat d’échange entre les trois composants :

- **Client C++** → **Serveur Go** (TCP, JSON ligne par ligne)
- **Serveur Go** → **Dashboard Next.js** (WebSocket, JSON)

Toute implémentation (Go, C++, Next.js) doit respecter ce schéma.

---

## 1. Transport

### 1.1 Client C++ → Serveur Go (TCP)

- **Encodage** : JSON uniquement, UTF-8.
- **Délimiteur** : un message = une ligne JSON terminée par `\n` (LF).
- Le serveur lit en mode « ligne par ligne » (bufio.Scanner ou équivalent).
- Pas de message binaire, pas de protobuf.

### 1.2 Serveur Go → Dashboard (WebSocket)

- **Encodage** : JSON, UTF-8.
- Chaque frame WebSocket (text) contient un seul objet JSON (pas de tableau de messages ; un message par frame).
- Le dashboard parse chaque frame comme un objet JSON.

---

## 2. Champs obligatoires (Client → Serveur)

Pour les événements **Client → Serveur** (TCP) :

- **Tous les types** : `type` (string), `room_id` (string, non vide), `timestamp` (number, Unix secondes).
- **Événements étudiant** (`student_joined`, `student_left`, `student_active`) : en plus, `student_id` (string, non vide).
- **Événements professeur** (`teacher_joined`, `teacher_left`) : en plus, `teacher_id` (string, non vide).

L’absence ou le type incorrect d’un champ obligatoire entraîne le rejet du message (voir section 5).

---

## 3. Événements Client C++ → Serveur Go (TCP)

Le client envoie l’un des types suivants (étudiants ou professeurs).

### 3.1 `student_joined`

Indique qu’un étudiant rejoint une salle.

```json
{
  "type": "student_joined",
  "student_id": "student-abc123",
  "room_id": "room-1",
  "timestamp": 1708455600
}
```

### 3.2 `student_left`

Indique qu’un étudiant quitte la salle.

```json
{
  "type": "student_left",
  "student_id": "student-abc123",
  "room_id": "room-1",
  "timestamp": 1708455660
}
```

### 3.3 `student_active`

Indique une activité (heartbeat / présence) de l’étudiant dans la salle.

```json
{
  "type": "student_active",
  "student_id": "student-abc123",
  "room_id": "room-1",
  "timestamp": 1708455620
}
```

**Contraintes** :

- `student_id` et `room_id` : chaînes non vides.
- `timestamp` : entier ≥ 0 (Unix seconds).
- Tout autre champ est ignoré par le serveur (extensibilité future).

### 3.4 `teacher_joined`

Indique qu’un professeur entre dans la salle. **Accepté uniquement si la salle est en mode `open`** : si un professeur est déjà présent et la salle est verrouillée, tout `teacher_joined` supplémentaire est rejeté (aucun second prof ne peut rejoindre une salle rouge). **Effet métier** : la salle passe en mode **salle rouge** (verrouillée) : plus aucune nouvelle entrée (étudiants ni professeurs) jusqu’à déverrouillage ; les personnes déjà présentes peuvent quitter (`student_left`, `teacher_left`) ou pour les élèves envoyer `student_active`.

```json
{
  "type": "teacher_joined",
  "teacher_id": "teacher-001",
  "room_id": "room-1",
  "timestamp": 1708455600,
  "duration_seconds": 600
}
```

| Champ               | Type   | Obligatoire | Description |
|---------------------|--------|-------------|-------------|
| `teacher_id`        | string | oui         | Identifiant unique du professeur. |
| `room_id`           | string | oui         | Identifiant de la salle. |
| `timestamp`         | number | oui         | Unix timestamp (secondes). |
| `duration_seconds`  | number | non         | Durée du blocage en secondes (défaut : 600 = 10 min). |

### 3.5 `teacher_left`

Indique qu’un professeur quitte la salle. La salle se déverrouille immédiatement et le timer de blocage est annulé (voir règles métier ci‑dessous).

```json
{
  "type": "teacher_left",
  "teacher_id": "teacher-001",
  "room_id": "room-1",
  "timestamp": 1708456200
}
```

### 3.6 Règles métier « salle rouge »

- **À la réception de `teacher_joined`** (salle ouverte) : la salle passe en `mode: "locked"`, `locked_until = now + duration_seconds` (défaut 600), `teacher = { teacher_id, joined_at }`. Le serveur diffuse immédiatement un `room_state` mis à jour.
- **Tant que la salle est verrouillée** :
  - Tout `student_joined` et tout `teacher_joined` sont **rejetés** (ignorés, logués). Aucune nouvelle entrée (étudiant ni professeur).
  - Les personnes déjà dans la salle restent. `student_left` et `student_active` sont traités normalement (les élèves peuvent sortir).
  - `teacher_left` reste accepté (le professeur déjà présent peut quitter).
- **Déverrouillage** : la salle repasse en `mode: "open"` (et `teacher` absent ou `null`) lorsque **au moins une** des conditions suivantes est vraie :
  - l’heure courante ≥ `locked_until` (fin du blocage temporisé), ou
  - le professeur présent envoie `teacher_left`.
- **Quand le professeur envoie `teacher_left`** : déverrouillage immédiat et **timer annulé (reset)**. Le serveur **vide la liste des étudiants** de cette salle (fin du cours — ils ne sont plus dans la salle). Il n’y a pas de blocage résiduel — si le prof part à 2 min et revient à 5 min, la salle est à nouveau ouverte et un nouveau `teacher_joined` crée un nouveau verrouillage (avec un nouveau `locked_until`).
- Après déverrouillage, étudiants et professeurs peuvent à nouveau rejoindre (`student_joined` et `teacher_joined` acceptés).

---

## 4. Événements Serveur Go → Dashboard (WebSocket)

Le serveur envoie au dashboard un seul type de message décrit ici.

### 4.1 `room_state`

État complet d’une salle : mode (ouverte / salle rouge), liste des professeurs présents, liste des étudiants et leur statut. Envoyé à chaque changement significatif et à la connexion d’un nouveau client WebSocket (snapshot initial).

**Champs obligatoires** :

| Champ       | Type   | Description |
|-------------|--------|-------------|
| `type`      | string | Toujours `"room_state"`. |
| `room_id`   | string | Identifiant de la salle. |
| `mode`      | string | `"open"` (entrées autorisées) ou `"locked"` (salle rouge : plus aucune entrée, ni étudiant ni professeur). |
| `students`  | array  | Liste des étudiants dans la salle (voir ci‑dessous). |
| `timestamp` | number | Unix timestamp (secondes) de la génération de l’état. |

**Champs optionnels** (présents lorsque la salle est verrouillée) :

| Champ          | Type   | Description |
|----------------|--------|-------------|
| `locked_until` | number | Unix timestamp (secondes) jusqu’auquel le blocage est actif. |
| `teacher`      | object | Le professeur présent dans la salle (celui dont l’entrée a verrouillé la salle). Absent ou `null` quand `mode === "open"`. |

**Structure de `teacher`** (quand salle verrouillée) :

| Champ        | Type   | Description |
|--------------|--------|-------------|
| `teacher_id` | string | Identifiant du professeur. |
| `joined_at`  | number | Unix timestamp d’entrée dans la salle. |

**Structure d’un élément de `students`** :

| Champ        | Type   | Description |
|--------------|--------|-------------|
| `student_id` | string | Identifiant de l’étudiant. |
| `status`     | string | Toujours `"active"` — seuls les étudiants présents dans la salle figurent dans la liste (voir ci‑dessous). |
| `last_seen`  | number | Dernier timestamp d’activité (Unix seconds). |

**Exemple — salle ouverte** :

```json
{
  "type": "room_state",
  "room_id": "room-1",
  "mode": "open",
  "students": [
    {
      "student_id": "student-abc123",
      "status": "active",
      "last_seen": 1708455620
    },
    {
      "student_id": "student-def456",
      "status": "active",
      "last_seen": 1708455600
    }
  ],
  "timestamp": 1708455665
}
```

**Exemple — salle rouge (cours bloqué)** :

```json
{
  "type": "room_state",
  "room_id": "room-1",
  "mode": "locked",
  "locked_until": 1708456200,
  "teacher": {
    "teacher_id": "teacher-001",
    "joined_at": 1708455600
  },
  "students": [
    {
      "student_id": "student-abc123",
      "status": "active",
      "last_seen": 1708455620
    }
  ],
  "timestamp": 1708455665
}
```

**Sémantique — liste `students`** :

La liste `students` ne contient **que les étudiants actuellement dans la salle**. Quand un étudiant envoie `student_left`, le serveur le **retire** de la liste (il n’est pas « idle », il n’est plus là). Quand le cours se termine (`teacher_left`), le serveur **vide** la liste des étudiants de cette salle (ou les retire tous). Pas de statut « idle » pour quelqu’un qui est parti.

Tant qu’il est dans la liste, l’étudiant est **`active`**. Le champ `last_seen` peut servir à l’affichage (optionnel), mis à jour à chaque `student_active`.

Résumé : join → dans la liste avec `active` ; leave ou fin du cours → **retiré de la liste**.

**Affichage dashboard** : `mode` pour la salle rouge, `teacher` et `locked_until` pour le prof et le timer. Les entrées de `students` sont les présents ; utiliser `last_seen` si besoin pour l’affichage.

---

## 5. Gestion des erreurs protocolaires

### 5.1 Côté serveur Go (réception TCP)

- **Message invalide** (JSON invalide, ligne mal formée) : le serveur logue l’erreur, ignore la ligne, et continue de traiter la connexion (pas de déconnexion du client).
- **Champ obligatoire manquant** (selon le type : `type`, `room_id`, `timestamp` ; plus `student_id` pour les événements étudiant, `teacher_id` pour les événements professeur) : log, ignorer le message, continuer.
- **`type` inconnu** : log, ignorer le message, continuer.
- **`type` valide mais valeurs incohérentes** (ex. `student_id` ou `teacher_id` vide) : log, ignorer le message, continuer.
- **`student_joined` ou `teacher_joined` alors que la salle est en mode `locked`** : log, ignorer le message (aucune entrée en salle rouge), continuer.

Le serveur **ne renvoie pas** d’ack/nack sur le TCP pour ces cas : le protocole est fire-and-forget du client vers le serveur. Les éventuels retours (ex. erreur applicative) pourront être définis dans une version ultérieure du protocole (optionnel).

### 5.2 Côté dashboard (réception WebSocket)

- **Frame non-JSON ou JSON invalide** : ignorer la frame, éventuellement log en dev.
- **`type` inconnu** : ignorer le message (extensibilité).
- **Champ obligatoire manquant dans `room_state`** : traiter comme état partiel ou ignorer le message (stratégie au choix du front, documentée dans le dashboard).

### 5.3 Résumé

| Situation              | Action côté serveur (TCP)     | Action côté dashboard (WS) |
|------------------------|-------------------------------|-----------------------------|
| JSON invalide          | Log, ignorer la ligne         | Ignorer la frame            |
| Champ obligatoire manquant | Log, ignorer le message   | Ignorer ou état partiel     |
| `type` inconnu         | Log, ignorer le message       | Ignorer le message          |

---

## 6. Résumé des types

| Direction        | `type`           | Description |
|------------------|------------------|-------------|
| C++ → Go (TCP)   | `student_joined` | Étudiant rejoint la salle. |
| C++ → Go (TCP)   | `student_left`   | Étudiant quitte la salle. |
| C++ → Go (TCP)   | `student_active` | Heartbeat / activité étudiant. |
| C++ → Go (TCP)   | `teacher_joined` | Professeur entre → salle rouge (blocage configurable, ex. 10 min). |
| C++ → Go (TCP)   | `teacher_left`   | Professeur quitte la salle → déverrouillage. |
| Go → Dashboard (WS) | `room_state`  | Snapshot : mode, teacher (objet unique si verrouillé), students, locked_until. |

---

## 7. Validation des implémentations

- **Go** : les adapters TCP et WebSocket doivent parser/produire uniquement ces types et champs ; les tests unitaires ou d’intégration peuvent valider les exemples de ce document.
- **C++** : le client doit envoyer uniquement des lignes JSON conformes aux exemples des sections 3.1–3.5.
- Ce document est la **source of truth** : toute divergence entre `docs/protocol.md` et le code doit être résolue en faveur de ce document (ou par mise à jour explicite du protocole ici).
