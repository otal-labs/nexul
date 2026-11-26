// The shell's menus open these, so they load on first open (warmed when idle), never with the shell.
export const loadPermissionsPanel = async () => {
  const [panel, header] = await Promise.all([import("@/components/access/PermissionsPanel"), import("@/components/access/PermissionsPanelHeader")]);
  return { PermissionsPanel: panel.PermissionsPanel, PermissionsPanelHeader: header.PermissionsPanelHeader };
};

export const loadChannelSettingsDialog = async () => (await import("@/components/chat/ChannelSettingsDialog")).ChannelSettingsDialog;
