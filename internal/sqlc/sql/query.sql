--------
-- indexes
--------

-- name: GetIndex :one
SELECT *
FROM indexes
WHERE id = ? LIMIT 1;

-- name: GetIndexByName :one
SELECT *
FROM indexes
WHERE name = ? LIMIT 1;

-- name: ListIndexes :many
SELECT *
FROM indexes
ORDER BY name;

-- name: CreateIndex :one
INSERT INTO indexes (name, description, path)
VALUES (?, ?, ?) RETURNING *;

-- name: UpdateIndex :exec
UPDATE indexes SET name = ?, description = ? WHERE id = ?;

-- name: DeleteIndex :exec
DELETE FROM indexes WHERE id = ?;

--------
-- documents
--------

-- name: GetDocument :one
SELECT *
FROM documents
WHERE id = ? LIMIT 1;

-- name: ListDocuments :many
SELECT *
FROM documents
ORDER BY created_at;

-- name: CreateDocument :one
INSERT INTO documents (index_id, filePath, fileType, fileSize)
VALUES (?, ?, ?, ?) RETURNING *;

-- name: UpdateDocument :exec
UPDATE documents SET filePath = ?, fileType = ?, fileSize = ? WHERE id = ?;

-- name: DeleteDocument :exec
DELETE FROM documents WHERE id = ?;

--------
-- chunks
--------

-- name: GetChunk :one
SELECT *
FROM chunks
WHERE id = ? LIMIT 1;

-- name: ListChunks :many
SELECT *
FROM chunks
ORDER BY start_offset;

-- name: CreateChunk :one
INSERT INTO chunks (document_id, start_offset, end_offset, content, context)
VALUES (?, ?, ?, ?, ?) RETURNING *;

-- name: UpdateChunk :exec
UPDATE chunks SET document_id = ?, start_offset = ?, end_offset = ?, content = ?, context = ? WHERE id = ?;

-- name: DeleteChunk :exec
DELETE FROM chunks WHERE id = ?;

--------
-- conversations
--------

-- name: GetConversation :one
SELECT *
FROM conversations
WHERE id = ? LIMIT 1;

-- name: ListConversations :many
SELECT *
FROM conversations
ORDER BY created_at;

-- name: CreateConversation :one
INSERT INTO conversations DEFAULT VALUES RETURNING *;

-- name: UpdateConversation :exec
UPDATE conversations SET updated_at = ? WHERE id = ?;

-- name: DeleteConversation :exec
DELETE FROM conversations WHERE id = ?;

--------
-- messages
--------

-- name: GetMessage :one
SELECT *
FROM messages
WHERE id = ? LIMIT 1;

-- name: ListMessages :many
SELECT *
FROM messages
ORDER BY created_at;

-- name: CreateMessage :one
INSERT INTO messages (conversation_id, ipv4_addr, user_agent, content)
VALUES (?, ?, ?, ?) RETURNING *;

-- name: UpdateMessage :exec
UPDATE messages SET conversation_id = ?, ipv4_addr = ?, user_agent = ?, content = ? WHERE id = ?;

-- name: DeleteMessage :exec
DELETE FROM messages WHERE id = ?;
