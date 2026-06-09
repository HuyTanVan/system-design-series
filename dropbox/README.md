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
| **Metadata** | `fileID` (pk), `name` (varchar), `size` (int), `uploadedBy` (userID), `mimeType` (varchar) |

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
- There are relationships between User, File, FileMetadata, so can choose PostgreSQL

### Where to store File (actual byte)?
#### Approach 1: Store in the primary server (Bad)
- Users upload files, then file contents as byte are stored in primary server while file's metadata is stored in database like PostgreSQL.
- Workflow: Users -> API gateway -> File Service(store File) -> PostgreSQL(store Metadata)
- Pros: Simple, fine for small application
- Cons: Won't scale well, bad secure.

#### Approach 2: Blob Storage (Good)
- Use an external storage system like AWS S3, Google Cloud Storage, ...
- Workflow: Users -> API gateway -> File Service 
                                    ├─> PostgreSQL (metadata)
                                    └─> S3 (files)                           
- Pros: Scale well
- Cons: Raise more complexity(need to handle cases when metadata is saved to DB, but failed to upload to Blob Storage) -> solved by transactional approach. Redundant upload(user -> backend, backend -> Blob Storage) -> solved by Approach 3.

#### Approach 3: Blob Storage direct upload.
- Allow the user to upload a file directly to Blob Storage using **presigned URLs**
> Presigned URLs is a URL that the user can use to upload the file directly to the Blob Storage service without going via backend.
- Workflow: Users -> API gateway -> File Service 
            |                        ├─> PostgreSQL (metadata)
            └─> S3 (files)                           
- Pros: handling large file transfers efficiently.

### 2. Users should be able to download files from any device.

#### Approach 1: Download through File Service(server) (bad)
- Workflow: User sends a download request, File Service downloads the file from S3, User downloads the file from File Service.
- Pros: dont think there is one
- Cons: Duplicate download(File Service <- S3, User <- File Service)        

#### Approach 2: Download directly from Blob storage (Good)
- Allow the user to download a file directly from Blob Storage using **presigned URLs**
- Pros: Fast, safe.
- Cons: Not optimal for users who are far away from Blob Storage server.
> Solution: Approach 3 using CDN to cache file

#### Approach 3: Download from CDN (optimization for Approach 2 -> Best)
- A CDN is a network of servers distributed across the globe that cache files and serve them to users from the server closest to them
- Pros: Fast for any user in any part of the world
- Cons: Expensive.

### 3. Users should be able to share files with other users and view the files shared with them.

#### Approach 1: Add a sharelist to Metadata(bad)
- a user expects to see their own files and files shared with them. Getting the list of their files is easy and can be optimized by using index on **uploadedBy**. However, getting the list of files shared with them is very slow(need to scan the **sharelist** of every file to see if user is in it)

#### Approach 2: Cache files shared for each user.
- user1 : [file1, file2, file3], user2: [file2, file5, file7], ...
- pros: fetching the Sharelist faster, simple key:value
- cons: raise a bit complexity(need to keep the Sharelist in cache in sync with the file metadata in database) -> solved by using a transaction(update both in cache and database the same)

#### Approach 3: Create a new database table for shared files.
- userID | fileID
- pros: simpler(no need to handle sync for map and database), query speed can be improved using indexing database.
- cons: slower than cache a bit
> Approach 2 and 3 is negotiable in different usecases, can be combined to bring the best solution.

---

### Future Improvements

---
