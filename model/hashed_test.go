package model

import (
	"testing"

	"gopkg.in/yaml.v3"
)

const (
	// plain password
	password = "Passw0rd"
	// echo -n Passw0rd | sha256sum
	sha256Password = "{sha256}ab38eadaeb746599f2c1ee90f8267f31f467347462764a24d71ac1843ee77fe3"
)

type TestHashedStruct struct {
	Password HashedString `yaml:"password"`
}

func TestHashedString(t *testing.T) {
	if SHA256Hash(password, true) != sha256Password {
		t.Fatalf("SHA256Hash returned wrong password")
	}

	hashed := NewHashedString(password)
	if !hashed.VerifyPlain(password) {
		t.Fatalf("NewHashedString returned wrong password")
	}

	hashed = NewHashedString(sha256Password)
	if !hashed.VerifyPlain(password) {
		t.Fatalf("NewHashedString returned wrong password")
	}
}

func TestYaml(t *testing.T) {
	var s1, s2 TestHashedStruct
	s1 = TestHashedStruct{
		Password: NewHashedString(password),
	}
	data, err := yaml.Marshal(s1)
	if err != nil {
		t.Fatal(err)
	}
	err = yaml.Unmarshal(data, &s2)
	if err != nil {
		t.Fatal(err)
	}
	if s1.Password != s2.Password || !s1.Password.VerifyPlain(password) || !s2.Password.VerifyPlain(password) {
		t.Fatalf("TestHashedStruct returned wrong password")
	}
}
