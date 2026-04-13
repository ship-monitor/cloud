package data

const (
	PermissionUserEdit    = "user.edit"
	PermissionUserList    = "user.list"
	PermissionUserBlock   = "user.block"
	PermissionUserGetByID = "user.get-by-id"

	PermissionOrganizationList             = "organization.list"
	PermissionOrganizationEdit             = "organization.edit"
	PermissionOrganizationGetByID          = "organization.get-by-id"
	PermissionOrganizationDelete           = "organization.delete"
	PermissionOrganizationInviteMemeber    = "organization.members.invite"
	PermissionOrganizationListMemebers     = "organization.members.list"
	PermissionOrganizationRemoveMemebers   = "organization.members.remove"
	PermissionOrganizationCreateRole       = "organization.roles.create"
	PermissionOrganizationEditRoles        = "organization.roles.edit"
	PermissionOrganizationDeleteRoles      = "organization.roles.delete"
	PermissionOrganizationConnectDevice    = "organization.devices.connect"
	PermissionOrganizationDisconnectDevice = "organization.devices.disconnect"
	PermissionOrganizationDeviceEditTags   = "organization.devices.edit-tags"
	PermissionOrganizationTagsEdit         = "organization.tags.edit"

	PermissionDeviceList   = "device.list"
	PermissionDeviceCreate = "device.create"
	PermissionDeviceEdit   = "device.edit"
	PermissionDeviceDelete = "device.delete"
)

func GetAllPermissions() []string {
	return []string{
		PermissionUserEdit,
		PermissionUserList,
		PermissionUserBlock,
		PermissionUserGetByID,

		PermissionOrganizationList,
		PermissionOrganizationEdit,
		PermissionOrganizationGetByID,
		PermissionOrganizationDelete,
		PermissionOrganizationInviteMemeber,
		PermissionOrganizationListMemebers,
		PermissionOrganizationRemoveMemebers,
		PermissionOrganizationCreateRole,
		PermissionOrganizationEditRoles,
		PermissionOrganizationDeleteRoles,
		PermissionOrganizationDeviceEditTags,
		PermissionOrganizationConnectDevice,
		PermissionOrganizationDisconnectDevice,
		PermissionOrganizationTagsEdit,
	}
}
