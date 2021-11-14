job "plugin-gluster-csi-controller" {
  datacenters = ["dc1"]

  group "controller" {
    task "plugin" {
      driver = "docker"

      config {
        image = "dragma/gluster-heketi-csi-driver"
        network_mode = "host"
        args = [
          "controller",
          "--endpoint=unix://csi/csi.sock",
          "--heketiurl=http://localhost:8080"
        ]
      }

      csi_plugin {
        id        = "glusterfs"
        type      = "controller"
        mount_dir = "/csi"
      }

      resources {
        cpu    = 100
        memory = 128
      }
    }
  }
}
