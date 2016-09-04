package main

import "testing"

var (
	usr1, usr2 User
	r1, r2, r3 Right
)

func init() {
	usr1 = User{
		Email:  "user1@test.tld",
		Groups: []Group{Group{Id: "group1"}, Group{Id: "group2"}}}

	usr2 = User{
		Email:  "user2@test.tld",
		Groups: []Group{Group{Id: "group2"}, Group{Id: "group3"}}}

	r1 = Right{
		All: "r",
		Groups: []EntityRight{
			EntityRight{Name: "group1", Rights: "rw"},
		},
		Users: []EntityRight{
			EntityRight{Name: "user2@test.tld", Rights: "rw"},
		},
	}

	// This one shall never exist
	r2 = Right{
		All: "rwa",
		Groups: []EntityRight{
			EntityRight{Name: "group1", Rights: "r"},
			EntityRight{Name: "group2", Rights: "r"},
			EntityRight{Name: "group3", Rights: "r"},
		},
	}

	r3 = Right{
		All: "r",
		Groups: []EntityRight{
			EntityRight{Name: "group4", Rights: "rw"},
		},
		Users: []EntityRight{
			EntityRight{Name: "user3@test.tld", Rights: "rwa"},
		},
	}
}

func assert(t *testing.T, e bool) {
	if !e {
		t.Fail()
	}
}

func TestRightsChecking(t *testing.T) {
	assert(t, GetRightType(&usr1, &r1) == AllowedToWrite)
	assert(t, GetRightType(&usr1, &r2) == AllowedToChangeRights)
	assert(t, GetRightType(&usr1, &r3) == AllowedToRead)
	assert(t, GetRightType(&usr2, &r1) == AllowedToWrite)
	assert(t, GetRightType(&usr2, &r2) == AllowedToChangeRights)
	assert(t, GetRightType(&usr2, &r3) == AllowedToRead)
}

func BenchmarkRightsChecking(b *testing.B) {
	for n := 0; n < b.N; n++ {
		GetRightType(&usr2, &r1)
	}
}
