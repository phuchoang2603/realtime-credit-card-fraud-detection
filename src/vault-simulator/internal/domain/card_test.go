package domain

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"testing"
)

func luhnCheckDigit(body string) string {
	for digit := 0; digit < 10; digit++ {
		if candidate := body + strconv.Itoa(digit); LuhnValid(candidate) {
			return strconv.Itoa(digit)
		}
	}
	return ""
}

func generatedNumber(prefix string, serial int) string {
	body := fmt.Sprintf("%s%011d", prefix, serial)
	return body + luhnCheckDigit(body)
}

type vectorFile struct {
	SyntheticKey string `json:"synthetic_key"`
	TestCards    []struct {
		Number   string `json:"number"`
		CardRef  string `json:"card_ref"`
		Behavior string `json:"behavior"`
		Expected string `json:"expected"`
	} `json:"test_cards"`
}

func TestAllowListMatchesIdentityVectors(t *testing.T) {
	raw, err := os.ReadFile("../../../../contracts/synthetic/identity-vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	var vectors vectorFile
	if err := json.Unmarshal(raw, &vectors); err != nil {
		t.Fatal(err)
	}
	if len(vectors.TestCards) == 0 {
		t.Fatal("no test_cards vectors")
	}
	for _, vector := range vectors.TestCards {
		t.Run(vector.CardRef, func(t *testing.T) {
			if !LuhnValid(vector.Number) {
				t.Fatalf("vector number is not Luhn valid")
			}
			card, ok := LookupCard(vector.Number)
			if !ok {
				t.Fatal("vector card is not allow-listed")
			}
			if card.Ref != vector.CardRef || string(card.Behavior) != vector.Behavior {
				t.Fatalf("got %+v, want ref %s behavior %s", card, vector.CardRef, vector.Behavior)
			}
			if got := Fingerprint([]byte(vectors.SyntheticKey), card.Ref); got != vector.Expected {
				t.Fatalf("fingerprint %s, want %s", got, vector.Expected)
			}
		})
	}
}

func TestLookupCardRanges(t *testing.T) {
	tests := []struct {
		name    string
		number  string
		wantOK  bool
		wantRef string
		wantBeh Behavior
	}{
		{"first ok", generatedNumber("4111", 0), true, "card-0005", BehaviorOK},
		{"last ok", generatedNumber("4111", 1999), true, "card-2004", BehaviorOK},
		{"first nsf", generatedNumber("4112", 0), true, "card-2005", BehaviorNSF},
		{"first issuer", generatedNumber("4113", 0), true, "card-4005", BehaviorIssuer},
		{"last invalid", generatedNumber("4114", 1999), true, "card-8004", BehaviorInvalid},
		{"serial beyond range", generatedNumber("4111", 2000), false, "", ""},
		{"unknown prefix", generatedNumber("4115", 0), false, "", ""},
		{"bad check digit", "4111000000000001", false, "", ""},
		{"short", "4111", false, "", ""},
		{"non digits", "4111-0000-0000-0000", false, "", ""},
		{"empty", "", false, "", ""},
		{"valid luhn outside allow-list", "5555555555554444", false, "", ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			card, ok := LookupCard(test.number)
			if ok != test.wantOK || card.Ref != test.wantRef || card.Behavior != test.wantBeh {
				t.Fatalf("got %+v ok=%v", card, ok)
			}
		})
	}
}

func TestEveryAllowListedNumberIsLuhnValidAndUnique(t *testing.T) {
	seen := map[string]string{}
	for _, r := range generatedRanges {
		for serial := range serialsPerBehavior {
			number := generatedNumber(r.prefix, serial)
			card, ok := LookupCard(number)
			if !ok || card.Behavior != r.behavior {
				t.Fatalf("%s serial %d not allow-listed with behavior %s", r.prefix, serial, r.behavior)
			}
			if previous, dup := seen[card.Ref]; dup {
				t.Fatalf("card ref %s reused (%s)", card.Ref, previous)
			}
			seen[card.Ref] = r.prefix
		}
	}
	for number := range namedCards {
		if !LuhnValid(number) {
			t.Fatalf("named card %s is not Luhn valid", number)
		}
	}
	if len(seen) != 4*serialsPerBehavior {
		t.Fatalf("got %d generated cards", len(seen))
	}
}
