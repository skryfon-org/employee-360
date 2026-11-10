package entity

import "testing"

func TestUser_FullName(t *testing.T) {
	tests := []struct {
		name      string
		firstName string
		lastName  string
		want      string
	}{
		{
			name:      "both names present",
			firstName: "Jane",
			lastName:  "Doe",
			want:      "Jane Doe",
		},
		{
			name:      "only first name",
			firstName: "Jane",
			lastName:  "",
			want:      "Jane",
		},
		{
			name:      "only last name",
			firstName: "",
			lastName:  "Doe",
			want:      "Doe",
		},
		{
			name:      "both empty",
			firstName: "",
			lastName:  "",
			want:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := User{FirstName: tt.firstName, LastName: tt.lastName}
			if got := u.FullName(); got != tt.want {
				t.Errorf("FullName() = %q, want %q", got, tt.want)
			}
		})
	}
}
