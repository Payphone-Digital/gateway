package validation

import (
	"fmt"
	"strings"
)

func DefaultMessage(field, tag string) string {
	field = strings.ToLower(field)

	switch tag {
	case "required":
		return fmt.Sprintf("%s tidak boleh kosong", field)
	case "email":
		return fmt.Sprintf("%s harus berupa alamat email yang valid", field)
	case "numeric":
		return fmt.Sprintf("%s harus berupa angka", field)
	case "min":
		return fmt.Sprintf("%s tidak memenuhi panjang atau nilai minimal", field)
	case "max":
		return fmt.Sprintf("%s melebihi panjang atau nilai maksimal", field)
	case "len":
		return fmt.Sprintf("%s harus memiliki panjang tertentu", field)
	case "gte":
		return fmt.Sprintf("%s harus lebih besar atau sama dengan nilai minimum", field)
	case "gt":
		return fmt.Sprintf("%s harus lebih besar dari nilai minimum", field)
	case "lte":
		return fmt.Sprintf("%s harus lebih kecil atau sama dengan nilai maksimum", field)
	case "lt":
		return fmt.Sprintf("%s harus lebih kecil dari nilai maksimum", field)
	case "eq":
		return fmt.Sprintf("%s harus sama dengan nilai yang ditentukan", field)
	case "ne":
		return fmt.Sprintf("%s tidak boleh sama dengan nilai yang ditentukan", field)
	case "url":
		return fmt.Sprintf("%s harus berupa URL yang valid", field)
	case "uuid":
		return fmt.Sprintf("%s harus berupa UUID yang valid", field)
	case "ip":
		return fmt.Sprintf("%s harus berupa alamat IP yang valid", field)
	case "ipv4":
		return fmt.Sprintf("%s harus berupa alamat IPv4 yang valid", field)
	case "ipv6":
		return fmt.Sprintf("%s harus berupa alamat IPv6 yang valid", field)
	case "alphanum":
		return fmt.Sprintf("%s hanya boleh berisi huruf dan angka", field)
	case "alpha":
		return fmt.Sprintf("%s hanya boleh berisi huruf", field)
	case "alphanumunicode":
		return fmt.Sprintf("%s hanya boleh berisi huruf dan angka unicode", field)
	case "alphaunicode":
		return fmt.Sprintf("%s hanya boleh berisi huruf unicode", field)
	case "boolean":
		return fmt.Sprintf("%s harus bernilai true atau false", field)
	case "contains":
		return fmt.Sprintf("%s harus mengandung substring tertentu", field)
	case "startswith":
		return fmt.Sprintf("%s harus diawali dengan substring tertentu", field)
	case "endswith":
		return fmt.Sprintf("%s harus diakhiri dengan substring tertentu", field)
	case "datetime":
		return fmt.Sprintf("%s harus berupa tanggal/waktu dengan format valid", field)
	case "oneof":
		return fmt.Sprintf("%s harus salah satu dari nilai yang diperbolehkan", field)
	case "base64":
		return fmt.Sprintf("%s harus berupa string base64 yang valid", field)
	case "hexadecimal":
		return fmt.Sprintf("%s harus berupa nilai heksadesimal", field)
	case "json":
		return fmt.Sprintf("%s harus berupa JSON yang valid", field)
	case "lowercase":
		return fmt.Sprintf("%s harus dalam huruf kecil semua", field)
	case "uppercase":
		return fmt.Sprintf("%s harus dalam huruf besar semua", field)
	case "excludes":
		return fmt.Sprintf("%s tidak boleh mengandung substring tertentu", field)
	case "excludesall":
		return fmt.Sprintf("%s tidak boleh mengandung karakter tertentu", field)
	case "excludesrune":
		return fmt.Sprintf("%s tidak boleh mengandung karakter unicode tertentu", field)
	default:
		return fmt.Sprintf("%s tidak valid", field)
	}
}
