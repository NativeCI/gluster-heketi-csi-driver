id = "mysql"
name = "mysql"
type = "csi"
plugin_id = "glusterfs"
capacity_min = "1G"
capacity_max = "1G"
capability {
    attachment_mode = "file-system"
    access_mode = "single-node-writer"
}