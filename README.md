# Glusterfs-Heketi CSI driver
The CSI (Container Storage Interface) is a universal way to provide storage for the applications.

This version of the plugin is based on the https://github.com/gluster/gluster-csi-driver which is currently deprecated and uses GD (gluster daemon) API to provision and manage volumes.

Instead of relying on deprecated components, I decided to stitch together the setup i have, which is glusterfs + heketi API.

Heketi is only responsible for creating and deleting volumes and the heavy lifting is done by glusterfs-fuse library to actually mount a provisioned volume.

## Example
I am currently using this driver for Hashicorp Nomad, which has full support of CSI plugins, the deployment folder contains example of that. It would not take much time to adapt that to be used with kubernetes as well, if anyone is interested PRs are welcome :)

## Why the image size is so big
Because I have first tested with alpine based image (didn't work) and then with amazonlinux (failed as well) and started receiving weird errors about missing devices, so I decided to match my host setup which centos 7.

If you have any suggestions on how I can reduce image size, I would be more than welcome to discuss them.
