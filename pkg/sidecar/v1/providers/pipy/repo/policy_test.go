package repo

import "testing"

func TestSetHTTPResponseBufferSize(t *testing.T) {
	tests := []struct {
		name  string
		input int
		want  int
	}{
		{name: "zero uses streaming default", input: 0, want: 1},
		{name: "negative uses streaming default", input: -1, want: 1},
		{name: "positive value is preserved", input: 4096, want: 4096},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf := &PipyConf{}
			if !conf.setHTTPResponseBufferSize(tt.input) {
				t.Fatal("first buffer size update was not reported")
			}
			if got := conf.Spec.HTTPResponseBufferSize; got != tt.want {
				t.Fatalf("buffer size = %d, want %d", got, tt.want)
			}
			if conf.setHTTPResponseBufferSize(tt.input) {
				t.Fatal("unchanged buffer size was reported as an update")
			}
		})
	}
}
