package validation

func CustomMessage(field string) map[string]string {
	var customValidationMessages = map[string]map[string]string{
		"Email": {
			"required": "email tidak boleh kosong",
			"email":    "email tidak valid",
		},
		"Phone": {
			"required": "nomor telepon tidak boleh kosong",
			"numeric":  "nomor telepon harus berupa angka",
		},
		"Password": {
			"required": "password tidak boleh kosong",
			"min":      "password minimal 6 karakter",
		},
		"Firstname": {
			"required": "firstname tidak boleh kosong",
		},
		"Lastname": {
			"required": "lastname tidak boleh kosong",
		},
	}
	return customValidationMessages[field]
}
