package eval

import "testing"

func TestMatchSuffixForms(t *testing.T) {
  dst := "function:Routes/Login2.go#Login"
  for _, truth := range []string{dst, "Login", "Login2.go#Login", "Routes/Login2.go#Login"} {
    if !match(dst, truth) {
      t.Errorf("%q should match %q", truth, dst)
    }
  }
  for _, truth := range []string{"gin2.go#Login", "Logi", "Register"} {
    if match(dst, truth) {
      t.Errorf("%q must not match %q", truth, dst)
    }
  }
}
