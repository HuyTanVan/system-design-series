# Dropbox

## Overview
Design a file storage and synchronization system (like **Dropbox**) where users can upload files, sync them across devices, and share them with others.

---

## Architecture Summary
<div style="margin-left:3rem">
    <img src="./images/architecture.png" alt="UI" width="1000">
</div>

---

## 1. Requirements

### Functional Requirements
1. Users should be able to **upload** and **dowload** files from any device(phone, computer, ...).
2. Users should be able to share files with other users and view the files shared with them.
3. Users should be able to sync files across devices

### Non-Functional Requirements
1. **High availability**: System should be **highly available**, prioritizing availability over consistency.
2. **Latency**: Upload/Dowload/Sync should be low latency(no fixed number of latency because it depends partly on user's network bandwith) -> as fast as the server can do.
3. **Capacity**: System should support large files as 50GB.
4. **Security & Fault Tolarance**: System should be secure and able to recover files if they are corrupted or lost.

---

## 2. Core Entities

| Entity | Fields |
|--------|--------|
| **User** | `userID` (pk) |
| **File** | `content` (byte) |
| **Metadata** | `fileID` (pk), `name` (varchar), `size` (int), `uploadedBy` (userID),`mimeType` (varchar) |

---

## 3. API Design

### Upload a file
```
POST /files
Body: { 
    File
    FileMetadata
}
```

### Download a file
```
GET /files/{fileID}
Response: File + FileMetadata
```

### Share a file
```
POST /files/{fileID}/share
Body: { 
    "users": []
}
```

---

## 4. Data Flow[Optional] + High Level Design

### A. Data Flow

#### i. WRITE flow
<div style="margin-left:3rem">
    <img src="./images/write-flow.png" alt="Write Flow" width="600">
</div>

#### ii. READ flow
<div style="margin-left:3rem">
    <img src="./images/read-flow.png" alt="Read Flow" width="600">
</div>

---

## 5. Deep Dives


### Where to store the FileMetadata?
- There are relationships between User, File, FileMetadata, so can choose relational database like PostgreSQL

### Where to store File (actual byte)?

#### Approach 1: Store in the primary server (Bad)
- Users upload files, then file contents as byte are stored in primary server while file's metadata is stored in database like PostgreSQL.
- Workflow: Users -> API gateway -> File Service(stores File) -> PostgreSQL(stores Metadata)
- Pros: Simple, fine for small application
- Cons: Doesn't scale well with file size; single point of failure for both compute and storage

#### Approach 2: Blob Storage via Backend (Good)
- Use an external object storage system like AWS S3, Google Cloud Storage, ...
- Workflow: Users -> API gateway -> File Service 
                                    ├─> PostgreSQL (stores metadata)
                                    └─> S3 (stores file bytes)                           
- Pros: Storage scales well because it is external.    
- Cons: Wastes bandwith and adds latency(Redundant uploads: client -> server -> S3); raises more complexity(need to handle cases when metadata is saved to DB, but failed to upload to Blob Storage) -> solved by transactional approach.-> solved by Approach 3.

#### Approach 3: Blob Storage direct upload via presigned URLs (Best)
- Allow the user to upload a file directly to Blob Storage using **presigned URLs**
- Workflow:
  - User -> File Service -> PostgreSQL (write metadata to DB, get and issue presigned URL to user)
  - User -> S3 (upload file directly using the presigned URL)
> Presigned URLs is a URL that the user can use to upload the file directly to the Blob Storage service without going via backend.                  
- Pros: Handles large file transfers efficiently; reduces backend workload, it only handles metadata.

### 2. Users should be able to download files from any device.

#### Approach 1: Download through File Service(server) (Bad)
- Workflow: User sends a download request, File Service downloads the file from S3, then serves it to the user.
- Cons: Redundant download(File Service <- S3, User <- File Service) -> Wastes bandwith and adds latency      

#### Approach 2: Download directly from Blob storage (Good)
- File Service issues a presigned URL, the client uses the URL to download directly from S3.
- Pros: Fast, no backend bottleneck..
- Cons: Not optimal for users who are far away from Blob Storage server(geographical issue).
> Solution: Approach 3 using CDN to cache file

#### Approach 3: Download from CDN (optimization for Approach 2 -> Best)
- Cache and serve file content at locations close to users.
- Pros: Fast for any user in anywhere in the world
- Cons: CDN is expensive

### 3. Users should be able to share files with other users and view the files shared with them.

#### Approach 1: Add a sharelist to Metadata(Bad)
- a user expects to see their own files and files shared with them. Getting the list of their files is easy and can be optimized by using index on **uploadedBy**. However, getting the list of files shared with them is very slow(need to scan the **sharelist** of every file to see if user is in it -> N+1 Problem)

#### Approach 2: Cache files shared for each user using in-memory hashmap.
- user1 : [file1, file2, file3], user2: [file2, file5, file7], ...
- Pros: fetches the Sharelist faster, simple key:value
- Cons: increases memory pressure at scale; adds a bit complexity(need to keep the Sharelist in cache in sync with the file metadata in database) -> solved by using a transaction(update both in cache and database the same)

#### Approach 3: Create a new database table for shared files.
- A table of `(userID, fileID)` pairs, indexed on `userID`.
- Pros: simpler(no need to handle sync for hashmap and database); scales with the database; query speed improved improved using indexing database.
- cons: slightly lower than inmemory cache
> Approaches 2 and 3 aren't mutually exclusive — a cache can sit in front of the table depending on read volume and scale.

### 3. Users can automatically sync files acress devices.
- Workflow: user(local) -sync-> remote(Blog storage), remote(Blog storage) -syn-> user(other devices)

- **Local -> Remote**: When a user updates a file on their local device, the changes needs to be synced with the remote server.
- Approach: client side sync agent.
- 1. Monitors the local Dropbox folder for changes using OS file system events(like Apple File System on IOS, FileSystemWatcher on Windows or FSEvents on macOS).
- 2. when any changes is detected, push the modified file to a queue.
- 3. Sends changes to the server along with updated metadate using upload API.
- 4. If 2 users edit the same file, it causes confict -> resolved by "last write wins" stategy(the most recent update will be saved)

- **Remote -> Local**: Other devices need to know what changes happend in the remote server as fast as possible, so they can pull down those changes
- **Approach 1**: polling
- Client periodically checks the server if there is any changes since last sync. The server queries the database to check if any files user is watching has a **updatedAt** timestamp that is newer than the last time they synced.
- Pros: Simple
- Cons: slower to detect changes; wastes bandthwith if nothing has changed.

- **Approach 2**: WebSocket
- Each client maintains an open connection to the server, the server pushes notifications when changes occur.
- Pros: Real-time update
- Cons: more complex to implement; higher server resource usage

- Approach 3: Hybrid (polling + Websocket)
- Websocket as a real-time sync agent: Each client maintains a single WebSocket connection to the server(one per device), any changes will be pushed to the server through this connection -> real-time sync for any file change.
- Polling as a reliable fallback: Websocket sometimes is not reliable, connection can drop and messages can be lost -> solved by Periodic polling(eg: every 5 minutes), The client periodically poll the server to catch any changes it have missed, ensuring eventual consistency even if Websocket connection is temporarily interupted.

### 4. How to handle large file upload?