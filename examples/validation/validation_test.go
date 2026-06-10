package main

import (
	"testing"

	govalidator "github.com/asaskevich/govalidator"
)

func validate(v any) error {
	_, err := govalidator.ValidateStruct(v)
	return err
}

// ---------- email ----------

func TestEmail_Valid(t *testing.T) {
	v := struct {
		V string `valid:"required,email"`
	}{V: "user@example.com"}
	if err := validate(v); err != nil {
		t.Fatalf("expected valid, got %v", err)
	}
}

func TestEmail_Invalid(t *testing.T) {
	v := struct {
		V string `valid:"required,email"`
	}{V: "not-an-email"}
	if err := validate(v); err == nil {
		t.Fatal("expected error")
	}
}

// ---------- url ----------

func TestURL_Valid(t *testing.T) {
	v := struct {
		V string `valid:"required,url"`
	}{V: "https://example.com"}
	if err := validate(v); err != nil {
		t.Fatalf("expected valid, got %v", err)
	}
}

func TestURL_Invalid(t *testing.T) {
	v := struct {
		V string `valid:"required,url"`
	}{V: "not a url"}
	if err := validate(v); err == nil {
		t.Fatal("expected error")
	}
}

// ---------- ip ----------

func TestIP_Valid(t *testing.T) {
	v := struct {
		V string `valid:"required,ip"`
	}{V: "192.168.1.1"}
	if err := validate(v); err != nil {
		t.Fatalf("expected valid, got %v", err)
	}
}

func TestIP_Invalid(t *testing.T) {
	v := struct {
		V string `valid:"required,ip"`
	}{V: "999.999.999.999"}
	if err := validate(v); err == nil {
		t.Fatal("expected error")
	}
}

// ---------- ipv4 ----------

func TestIPv4_Valid(t *testing.T) {
	v := struct {
		V string `valid:"required,ipv4"`
	}{V: "10.0.0.1"}
	if err := validate(v); err != nil {
		t.Fatalf("expected valid, got %v", err)
	}
}

func TestIPv4_Invalid(t *testing.T) {
	v := struct {
		V string `valid:"required,ipv4"`
	}{V: "::1"}
	if err := validate(v); err == nil {
		t.Fatal("expected error")
	}
}

// ---------- ipv6 ----------

func TestIPv6_Valid(t *testing.T) {
	v := struct {
		V string `valid:"required,ipv6"`
	}{V: "::1"}
	if err := validate(v); err != nil {
		t.Fatalf("expected valid, got %v", err)
	}
}

func TestIPv6_Invalid(t *testing.T) {
	v := struct {
		V string `valid:"required,ipv6"`
	}{V: "192.168.1.1"}
	if err := validate(v); err == nil {
		t.Fatal("expected error")
	}
}

// ---------- mac ----------

func TestMAC_Valid(t *testing.T) {
	v := struct {
		V string `valid:"required,mac"`
	}{V: "01:23:45:67:89:ab"}
	if err := validate(v); err != nil {
		t.Fatalf("expected valid, got %v", err)
	}
}

func TestMAC_Invalid(t *testing.T) {
	v := struct {
		V string `valid:"required,mac"`
	}{V: "not-a-mac"}
	if err := validate(v); err == nil {
		t.Fatal("expected error")
	}
}

// ---------- alpha ----------

func TestAlpha_Valid(t *testing.T) {
	v := struct {
		V string `valid:"required,alpha"`
	}{V: "Hello"}
	if err := validate(v); err != nil {
		t.Fatalf("expected valid, got %v", err)
	}
}

func TestAlpha_Invalid(t *testing.T) {
	v := struct {
		V string `valid:"required,alpha"`
	}{V: "Hello123"}
	if err := validate(v); err == nil {
		t.Fatal("expected error")
	}
}

// ---------- numeric ----------

func TestNumeric_Valid(t *testing.T) {
	v := struct {
		V string `valid:"required,numeric"`
	}{V: "5551234567"}
	if err := validate(v); err != nil {
		t.Fatalf("expected valid, got %v", err)
	}
}

func TestNumeric_Invalid(t *testing.T) {
	v := struct {
		V string `valid:"required,numeric"`
	}{V: "555-123-4567"}
	if err := validate(v); err == nil {
		t.Fatal("expected error")
	}
}

// ---------- alphanum ----------

func TestAlphanum_Valid(t *testing.T) {
	v := struct {
		V string `valid:"required,alphanum"`
	}{V: "abc123"}
	if err := validate(v); err != nil {
		t.Fatalf("expected valid, got %v", err)
	}
}

func TestAlphanum_Invalid(t *testing.T) {
	v := struct {
		V string `valid:"required,alphanum"`
	}{V: "abc-123!"}
	if err := validate(v); err == nil {
		t.Fatal("expected error")
	}
}

// ---------- hexcolor ----------

func TestHexColor_Valid(t *testing.T) {
	v := struct {
		V string `valid:"required,hexcolor"`
	}{V: "#ff0033"}
	if err := validate(v); err != nil {
		t.Fatalf("expected valid, got %v", err)
	}
}

func TestHexColor_Invalid(t *testing.T) {
	v := struct {
		V string `valid:"required,hexcolor"`
	}{V: "red"}
	if err := validate(v); err == nil {
		t.Fatal("expected error")
	}
}

// ---------- creditcard ----------

func TestCreditCard_Valid(t *testing.T) {
	v := struct {
		V string `valid:"required,creditcard"`
	}{V: "4111111111111111"}
	if err := validate(v); err != nil {
		t.Fatalf("expected valid, got %v", err)
	}
}

func TestCreditCard_Invalid(t *testing.T) {
	v := struct {
		V string `valid:"required,creditcard"`
	}{V: "1234567890"}
	if err := validate(v); err == nil {
		t.Fatal("expected error")
	}
}

// ---------- base64 ----------

func TestBase64_Valid(t *testing.T) {
	v := struct {
		V string `valid:"required,base64"`
	}{V: "SGVsbG8gV29ybGQ="}
	if err := validate(v); err != nil {
		t.Fatalf("expected valid, got %v", err)
	}
}

func TestBase64_Invalid(t *testing.T) {
	v := struct {
		V string `valid:"required,base64"`
	}{V: "not base64!!!"}
	if err := validate(v); err == nil {
		t.Fatal("expected error")
	}
}

// ---------- latitude ----------

func TestLatitude_Valid(t *testing.T) {
	v := struct {
		V string `valid:"required,latitude"`
	}{V: "40.7128"}
	if err := validate(v); err != nil {
		t.Fatalf("expected valid, got %v", err)
	}
}

func TestLatitude_Invalid(t *testing.T) {
	v := struct {
		V string `valid:"required,latitude"`
	}{V: "91.0"}
	if err := validate(v); err == nil {
		t.Fatal("expected error")
	}
}

// ---------- longitude ----------

func TestLongitude_Valid(t *testing.T) {
	v := struct {
		V string `valid:"required,longitude"`
	}{V: "-74.0060"}
	if err := validate(v); err != nil {
		t.Fatalf("expected valid, got %v", err)
	}
}

func TestLongitude_Invalid(t *testing.T) {
	v := struct {
		V string `valid:"required,longitude"`
	}{V: "181.0"}
	if err := validate(v); err == nil {
		t.Fatal("expected error")
	}
}

// ---------- port ----------

func TestPort_Valid(t *testing.T) {
	v := struct {
		V string `valid:"required,port"`
	}{V: "8080"}
	if err := validate(v); err != nil {
		t.Fatalf("expected valid, got %v", err)
	}
}

func TestPort_Invalid(t *testing.T) {
	v := struct {
		V string `valid:"required,port"`
	}{V: "99999"}
	if err := validate(v); err == nil {
		t.Fatal("expected error")
	}
}

// ---------- semver ----------

func TestSemver_Valid(t *testing.T) {
	v := struct {
		V string `valid:"required,semver"`
	}{V: "1.2.3"}
	if err := validate(v); err != nil {
		t.Fatalf("expected valid, got %v", err)
	}
}

func TestSemver_Invalid(t *testing.T) {
	v := struct {
		V string `valid:"required,semver"`
	}{V: "v1.2"}
	if err := validate(v); err == nil {
		t.Fatal("expected error")
	}
}

// ---------- optional fields ----------

func TestOptional_Empty(t *testing.T) {
	v := struct {
		V string `valid:"optional,email"`
	}{V: ""}
	if err := validate(v); err != nil {
		t.Fatalf("expected optional empty to pass, got %v", err)
	}
}

func TestOptional_ValidValue(t *testing.T) {
	v := struct {
		V string `valid:"optional,email"`
	}{V: "a@b.com"}
	if err := validate(v); err != nil {
		t.Fatalf("expected valid, got %v", err)
	}
}

func TestOptional_InvalidValue(t *testing.T) {
	v := struct {
		V string `valid:"optional,email"`
	}{V: "bad"}
	if err := validate(v); err == nil {
		t.Fatal("expected error for optional field with invalid non-empty value")
	}
}
