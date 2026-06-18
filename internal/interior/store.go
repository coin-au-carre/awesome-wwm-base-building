package interior

import "ruby/internal/jsonstore"

func LoadFile(path string) ([]Interior, error) { return jsonstore.Load[Interior](path) }
func SaveFile(path string, interiors []Interior) error { return jsonstore.Save(path, interiors) }
