package main

type RightType int

const (
	Nothing RightType = iota
	AllowedToRead
	AllowedToWrite
	AllowedToChangeRights
)

type Right struct {
	All    string        `bson:"all" json:"all,omitempty"`
	Groups []EntityRight `bson:"groups" json:"groups,omitempty"`
	Users  []EntityRight `bson:"users" json:"users,omitempty"`
}

type EntityRight struct {
	Name   string `bson:"name" json:"name,omitempty"`
	Rights string `bson:"rights" json:"rights,omitempty"`
}

func RightStringToRightType(str string) RightType {
	switch str {
	case "n":
		return Nothing
	case "r":
		return AllowedToRead
	case "rw":
		return AllowedToWrite
	case "rwa":
		return AllowedToChangeRights
	}

	return Nothing
}

func UserBelongsToGroup(u *User, g string) bool {
	for _, group := range u.Groups {
		if group == g {
			return true
		}
	}

	return false
}

func GetRightType(u *User, r *Right) RightType {
	if u.IsAdmin != nil {
		if *u.IsAdmin {
			return AllowedToChangeRights
		}
	}

	if r == nil {
		// WARNING: this is the default policy when there is NO rights set
		return AllowedToWrite
	}

	result := RightStringToRightType(r.All)

	if result == AllowedToChangeRights {
		// Stop here as we are returning the most permissive right
		return result
	}

	for _, er := range r.Users {
		// TODO: by user ID instead
		if er.Name == u.Email {
			rights := RightStringToRightType(er.Rights)

			if rights > result {
				if rights == AllowedToChangeRights {
					return rights
				}
				result = rights
			}
		}
	}

	for _, er := range r.Groups {
		// TODO: by group ID instead
		if UserBelongsToGroup(u, er.Name) {
			rights := RightStringToRightType(er.Rights)
			if rights > result {
				result = rights
			}
		}
	}

	return result
}
