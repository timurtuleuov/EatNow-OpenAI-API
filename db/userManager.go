package db

// func initUserTable() error {

// 	tableScript :=
// 		`CREATE TABLE users (
// 		id UUID PRIMARY KEY DEFAULT gen_random_uuid(), -- уникальный ID
// 		name VARCHAR(100) NOT NULL,
// 		password VARCHAR(255) NOT NULL, -- хэш пароля
// 		email VARCHAR(255) UNIQUE NOT NULL,
// 		platform VARCHAR(50), -- например, 'android', 'ios', 'web'
// 		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
// 		updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL
// 	);
// 	`
// }
