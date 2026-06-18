package blueprint

import "ruby/internal/jsonstore"

func LoadFile(path string) ([]Blueprint, error) { return jsonstore.Load[Blueprint](path) }
func SaveFile(path string, blueprints []Blueprint) error { return jsonstore.Save(path, blueprints) }
