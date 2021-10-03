# Glusterfs-Heketi CSI driver
![CSI](./assets/csi.png)
The CSI (Container Storage Interface) is a universal way to provide storage for the applications.

This version of the plugin is based on the https://github.com/gluster/gluster-csi-driver which is currently deprecated and uses GD (gluster daemon) API to provision and manage volumes.

Instead of relying on deprecated components, I decided to stitch together the setup i have, which is glusterfs + heketi API.

Heketi is only responsible for creating and deleting volumes and the heavy lifting is done by glusterfs-fuse library to actually mount a provisioned volume.

## Example
I am currently using this driver for Hashicorp Nomad, which has full support of CSI plugins, the deployment folder contains example of that. It would not take much time to adapt that to be used with kubernetes as well, if anyone is interested PRs are welcome :)

Create the controller file first:
```hcl
job "plugin-gluster-csi-controller" {
  datacenters = ["dc1"]

  group "controller" {
    task "plugin" {
      driver = "docker"

      config {
        image = "dragma/gluster-heketi-csi-driver"

        args = [
          "controller",
          "--endpoint=unix://csi/csi.sock",
          "--v=5",
          "--heketiurl=http://localhost:8080"
        ]
      }

      csi_plugin {
        id        = "glusterfs"
        type      = "controller"
        mount_dir = "/csi"
      }

      resources {
        cpu    = 500
        memory = 256
      }
    }
  }
}
```

It mostly contains the defaults from Nomad documentation (https://learn.hashicorp.com/tutorials/nomad/stateful-workloads-csi-volumes?in=nomad/stateful-workloads).

Notes:
- Add the appropriate heketi url 
- If you use authentication for heketi also customise the --username and --heketisecret

Run the controller:
```
nomad job run controller.nomad
```

After that create the nodes.nomad file:
```hcl
job "plugin-gluster-csi-nodes" {
  datacenters = ["dc1"]

  # you can run node plugins as service jobs as well, but this ensures
  # that all nodes in the DC have a copy.
  type = "system"

  group "nodes" {
    task "plugin" {
      driver = "docker"

      config {
        image = "dragma/gluster-heketi-csi-driver"

        args = [
          "nodeserver",
          "--endpoint=unix://csi/csi.sock",
          "--nodeid=${node.unique.id}"
        ]

        # node plugins must run as privileged jobs because they
        # mount disks to the host
        privileged = true
      }

      csi_plugin {
        id        = "glusterfs"
        type      = "node"
        mount_dir = "/csi"
      }

      resources {
        cpu    = 500
        memory = 256
      }
    }
  }
}
```

Run the csi nodes:
```
nomad job run nodes.nomad
```

Finally make the following volume.hcl file:
```hcl
type = "csi"
id = "mysql"
name = "mysql"
plugin_id = "glusterfs"
capacity_min = "5G"
capacity_max = "5G"
capability {
  access_mode = "multi-node-multi-writer"
  attachment_mode = "file-system"
}
```
And run:
```
nomad volume create volume.hcl
```
This will provision the volume which you verify by running:
```
nomad volume status mysql
ID                   = mysql
Name                 = mysql
External ID          = 961f112001711b55f0cff982c16b4f02
Plugin ID            = glusterfs
Provider             = org.gluster.glusterfs
Version              = 1.0.0
Schedulable          = true
Controllers Healthy  = 1
Controllers Expected = 1
Nodes Healthy        = 1
Nodes Expected       = 1
Access Mode          = multi-node-multi-writer
Attachment Mode      = file-system
Mount Options        = <none>
Namespace            = default
```

Finally make the mysql.nomad file which will use the volume created:
```hcl
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
      access_mode     = "multi-node-multi-writer"
    }

    network {
      port "db" {
        static = 3306
      }
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
        ports = ["db"]
      }

      resources {
        cpu    = 100
        memory = 100
      }

      service {
        name = "mysql-server"
        port = "db"

        check {
          type     = "tcp"
          interval = "10s"
          timeout  = "2s"
        }
      }
    }
  }
}
```

Run the server:
```
nomad job run mysql.nomad
```

You can connect to the mysql server from the host:
```
mysql -h 127.0.0.1 -u web -p -D itemcollection
```

## Contributing
If you have any issues or would like to contribute, feel free to open an issue or PR.

