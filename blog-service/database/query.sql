-- name: ListPosts :many
SELECT * FROM posts
ORDER BY id DESC;

-- name: GetPost :one
SELECT * FROM posts
WHERE id = ?;


-- name: CreatePost :execresult
INSERT INTO posts (title, image, content, created_at, updated_at)
VALUES (?, ?, ?, NOW(), NOW());

-- name: UpdatePost :exec
UPDATE posts
SET title = ?, image = ?, content = ?, updated_at = NOW()
WHERE id = ?;

-- name: DeletePost :exec
DELETE FROM posts
WHERE id = ?;
