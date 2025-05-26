package domain

// deviceFilter holds “device_id” and “device_name” filters.  When non-nil,
// each is turned into a condition on the “devices” table.
//
//	schema:"device_id"      → matches URL param "device_id"
//	qrstr:"devices.id IN (?)" → SQL snippet (IN can accept a slice)
//	preload:"devices"       → GORM association to Preload
type DeviceFilter struct {
	IDs   *[]uint   `schema:"device_id"   qrstr:"devices.id IN (?)"   preload:"Devices"`
	Names *[]string `schema:"device_name" qrstr:"devices.name IN (?)" preload:"Devices"`
}

// GroupFilter holds nested group → permission filters.
//
//	schema:"group_name"      → matches URL param "group_name"
//	qrstr:"groups.name = ?"   → SQL snippet
//	preload:"Groups"          → GORM many2many association on User
//
// And the nested Permissions sub‐struct has its own tags:
//
//	schema:"permission"       → matches URL param "permission"
//	qrstr:"permissions.code=?" → SQL snippet on the “permissions” table
//	preload:"Groups.Permissions" → nested association path
type PermissionFilter struct {
	Codes *[]string `schema:"permission" qrstr:"permissions.code = ?" preload:"Groups.Permissions"`
}

type GroupFilter struct {
	Names       *[]string        `schema:"group_name" qrstr:"groups.name IN (?)" preload:"Groups"`
	Permissions PermissionFilter // no schema tag here—its own fields have tags
}

// Finally, the top‐level UserFilter.  It can have a “Name” filter,
// and also embed deviceFilter and GroupFilter.  The “Name” field
// could be a column on “users” (so exact “users.name = ?”).
type UserFilter struct {
	Name   *string      `schema:"user_name" qrstr:"users.name = ?"`
	Device DeviceFilter // filters on “Devices”
	Group  GroupFilter  // filters on “Groups” and nested “Groups.Permissions”
}

type UserFilterV3 struct {
	Name *string `schema:"user_name" qrstr:"users.name = ?"`
	// filters on “devices”
	DeviceIDs   *[]uint   `schema:"device_id"   qrstr:"devices.id IN (?)"   preload:"Devices"`
	DeviceNames *[]string `schema:"device_name" qrstr:"devices.name IN (?)" preload:"Devices"`
	// filters on “Groups” and nested “Groups.Permissions”
	GroupNames *[]string `schema:"group_name" qrstr:"groups.name IN (?)" preload:"Groups"`
	//filters on “permissions” table
	PermissionCodes *[]string `schema:"permission" qrstr:"permissions.code = ?" preload:"Groups.Permissions"`
}

type Devices struct {
	ID   uint   `gorm:"primaryKey" schema:"device_id"`
	Name string `gorm:"unique" schema:"device_name"`
}

type User struct {
	ID      uint `gorm:"primaryKey"`
	Name    string
	Devices []Devices `gorm:"many2many:user_devices;"`
	Groups  []Groups  `gorm:"many2many:user_groups;"`
}

type Groups struct {
	ID          uint         `gorm:"primaryKey" schema:"group_id"`
	Name        string       `gorm:"unique" schema:"group_name"`
	Permissions []Permission `gorm:"many2many:group_permissions;"`
}

type Permission struct {
	ID   uint   `gorm:"primaryKey" schema:"permission_id"`
	Code string `gorm:"unique" schema:"permission_code"`
}
