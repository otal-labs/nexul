-- name: CreateCategory :exec
INSERT INTO categories (id, project_id, name, position, color, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: GetCategory :one
SELECT * FROM categories WHERE id = ?;

-- name: ListCategoriesByProject :many
SELECT * FROM categories WHERE project_id = ? ORDER BY position, id;

-- name: ListCategories :many
SELECT * FROM categories ORDER BY project_id, position, id;

-- name: UpdateCategory :execrows
UPDATE categories SET name = ?, color = ?, updated_at = ? WHERE id = ?;

-- name: DeleteCategory :execrows
DELETE FROM categories WHERE id = ?;

-- name: ReorderCategoryPosition :execrows
UPDATE categories SET position = ?, updated_at = ? WHERE id = ? AND project_id = ?;

-- name: CountCategoryTickets :one
SELECT COUNT(*) FROM tickets WHERE category_id = ?;
