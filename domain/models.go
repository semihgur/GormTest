package domain

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
