package hw10programoptimization

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strings"
)

// User описывает формат записи во входном потоке.
type User struct {
	ID       int
	Name     string
	Username string
	Email    string
	Phone    string
	Password string
	Address  string
}

type DomainStat map[string]int

// maxLineSize ограничивает длину строки входных данных.
const maxLineSize = 64 * 1024

// emailKey — префикс поля, значение которого извлекается из записи.
var emailKey = []byte(`"Email":"`)

// GetDomainStat считает домены email-адресов из потока JSON-строк,
// отбирая адреса в зоне domain. Поток читается построчно, поэтому объём
// потребляемой памяти не зависит от размера входных данных.
func GetDomainStat(r io.Reader, domain string) (DomainStat, error) {
	result := make(DomainStat)
	suffix := []byte("." + domain)

	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, maxLineSize), maxLineSize)

	for scanner.Scan() {
		host, ok := domainOf(scanner.Bytes())
		if !ok || !bytes.HasSuffix(host, suffix) {
			continue
		}
		result[toLower(host)]++
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("get users error: %w", err)
	}

	return result, nil
}

// domainOf возвращает часть поля Email после '@', не копируя данные.
// Второе значение равно false, если поле отсутствует или адрес некорректен.
func domainOf(line []byte) ([]byte, bool) {
	i := bytes.Index(line, emailKey)
	if i < 0 {
		return nil, false
	}
	value := line[i+len(emailKey):]

	end := bytes.IndexByte(value, '"')
	if end < 0 {
		return nil, false
	}
	value = value[:end]

	at := bytes.LastIndexByte(value, '@')
	if at < 0 {
		return nil, false
	}
	return value[at+1:], true
}

// toLower приводит домен к нижнему регистру, выделяя память только под
// ключ карты.
func toLower(host []byte) string {
	for _, c := range host {
		if c >= 'A' && c <= 'Z' {
			return strings.ToLower(string(host))
		}
	}
	return string(host)
}
