package toml

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/davecgh/go-spew/spew"
)

var (
	wanted        = "----- Wanted -----"
	wantedErr     = "----- Wanted Err -----"
	actual        = "----- Actual -----"
	actualErr     = "----- Actual Err -----"
	divider       = strings.Repeat("-", len(wanted))
	divErr        = strings.Repeat("-", len(wantedErr))
	testResFmt    = "\n" + wanted + "\n%s\n" + actual + "\n%s\n" + divider + "\n"
	testResErrFmt = "\n" + wantedErr + "\n%s\n" + actualErr + "\n%s\n" + divErr + "\n"
)

type test struct {
	Name        string
	Unmarshaled any
	Marshaled   []byte
	wantError   error
}

type TestGeets struct {
	Geeters   string `toml:"geeters"`
	YeetIndex []int  `toml:"yeet_index"`
}
type TestMcGeeParameters struct {
	Username string     `toml:"username"`
	SkipTag  string     `toml:"-"`
	SubGeet  *TestGeets `toml:"sub_geet"`
}
type TestYeetersonParameters struct {
	ServerName string              `toml:"server_name"`
	DenyList   []string            `toml:"deny_list"`
	PortNumber int                 `toml:"port_number"`
	YeetMode   bool                `toml:"yeet_mode"`
	McGee      TestMcGeeParameters `toml:"mcgeet"`
	unexported string              `toml:"unexported"` //golint:unused
}
type parameters struct {
	Yeeterson TestYeetersonParameters `toml:"yeet"`
	McGee     TestMcGeeParameters     `toml:"mcgee"`
	SkipMe    string                  `toml:"-"`

	skipMe string `toml:"skip_me"` //golint:unused
}

var simpleYeeterson = TestYeetersonParameters{
	ServerName: "yeeterson",
	DenyList:   []string{"yeet", "yeeterson", "yeeterson.com"},
	PortNumber: 8080,
	YeetMode:   true,
}

var (
	test1 = test{
		Name: "simple",
		Unmarshaled: parameters{
			Yeeterson: simpleYeeterson,
			McGee: TestMcGeeParameters{
				Username: "mcgee",
			},
		},
		Marshaled: []byte(`[yeet]
server_name = "yeeterson"
deny_list = ["yeet", "yeeterson", "yeeterson.com"]
port_number = 8080
yeet_mode = true

[mcgee]
username = "mcgee"`),
	}

	test2 = test{
		Name: "with empty string, negative number, spaced strings, punctuation",
		Unmarshaled: parameters{
			Yeeterson: TestYeetersonParameters{
				ServerName: "",
				DenyList:   []string{"yeet it."},
				PortNumber: -5,
				YeetMode:   false,
			},
			McGee: TestMcGeeParameters{
				Username: "the yeet guy",
				SkipTag:  "skip me",
			},
			SkipMe: "skip me",
		},
		Marshaled: []byte(`[yeet]
server_name = ""
deny_list = ["yeet it."]
port_number = -5
yeet_mode = false

[mcgee]
username = "the yeet guy"`),
	}

	yeetersonWithChild = TestYeetersonParameters{
		ServerName: simpleYeeterson.ServerName,
		DenyList:   simpleYeeterson.DenyList,
		PortNumber: simpleYeeterson.PortNumber,
		YeetMode:   simpleYeeterson.YeetMode,
		McGee: TestMcGeeParameters{
			Username: "Yeeterson McGeeterson",
		},
	}

	test3 = test{
		Name: "with sub-structs",
		Unmarshaled: parameters{
			Yeeterson: yeetersonWithChild,
			McGee: TestMcGeeParameters{
				Username: "mcgee",
			},
		},
		Marshaled: []byte(`[yeet]
server_name = "yeeterson"
deny_list = ["yeet", "yeeterson", "yeeterson.com"]
port_number = 8080
yeet_mode = true

[yeet.mcgeet]
username = "Yeeterson McGeeterson"

[mcgee]
username = "mcgee"`),
	}

	test4 = test{
		Name: "with empty structs",
		Unmarshaled: parameters{
			Yeeterson: TestYeetersonParameters{},
			McGee: TestMcGeeParameters{
				Username: "mcgeets",
				SubGeet:  &TestGeets{},
			},
		},
		Marshaled: []byte(`[mcgee]
username = "mcgeets"`),
	}

	test5 = test{
		Name: "with pointer struct",
		Unmarshaled: parameters{
			Yeeterson: TestYeetersonParameters{},
			McGee: TestMcGeeParameters{
				Username: "geetsies",
				SubGeet:  &TestGeets{Geeters: "geets", YeetIndex: []int{1, 2, 3}},
			},
		},
		Marshaled: []byte(`[mcgee]
username = "geetsies"

[mcgee.sub_geet]
geeters = "geets"
yeet_index = [1, 2, 3]`),
	}

	tests = []test{test1, test2, test3, test4, test5}
)

//nolint:funlen
func TestMarshalTOML(t *testing.T) {
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			output, err := MarshalTOML(tt.Unmarshaled)
			if !errors.Is(err, tt.wantError) {
				errWantString := "nil"
				if tt.wantError != nil {
					errWantString = tt.wantError.Error()
				}
				t.Errorf(testResErrFmt, errWantString, err)
			}
			if string(output) != string(tt.Marshaled) {
				t.Errorf(testResFmt, tt.Marshaled, output)
			}
		})
	}
}

func TestUnmarshalTOML(t *testing.T) {
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			output := new(parameters)
			err := UnmarshalTOML(tt.Marshaled, output)
			if !errors.Is(err, tt.wantError) {
				errWantString := "nil"
				if tt.wantError != nil {
					errWantString = tt.wantError.Error()
				}
				t.Errorf(testResErrFmt, errWantString, err)
			}
			if !reflect.DeepEqual(*output, tt.Unmarshaled) {
				t.Errorf(testResFmt, tt.Unmarshaled, spew.Sdump(*output))
			}
		})
	}
}
