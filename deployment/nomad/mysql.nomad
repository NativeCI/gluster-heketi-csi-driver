job "mysql-server" {
  datacenters = ["dc1"]
  type        = "service"

  group "mysql-server" {
    count = 1

    volume "mysql" {
      type      = "csi"
      read_only = false
      source    = "mysql"
      attachment_mode = "file-system"
      access_mode = "single-node-writer"
    }
    restart {
      attempts = 10
      interval = "5m"
      delay    = "25s"
      mode     = "delay"
    }

    task "mysql-server" {
      driver = "docker"

      volume_mount {
        volume      = "mysql"
        destination = "/srv"
        read_only   = false
      }

      env {
        MYSQL_ROOT_PASSWORD = "password"
      }

      config {
        image = "hashicorp/mysql-portworx-demo:latest"
        args  = ["--datadir", "/srv/mysql"]
      }

      resources {
        cpu    = 300
        memory = 512
      }
    }
  }
}
