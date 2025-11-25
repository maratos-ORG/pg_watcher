
# restart VMWare-FUusion
```bash
sudo "/Applications/VMware Fusion.app/Contents/Library/services/services.sh" --stop
sudo "/Applications/VMware Fusion.app/Contents/Library/services/services.sh" --start
```

# Install box if not exist 
```bash
vagrant box list
vagrant box add bento/ubuntu-22.04 --provider vmware_desktop
```

# Lunch Vagrant 
```bash
[[ $(uname -m) == "arm64" ]] && export VAGRANT_VAGRANTFILE="Vagrantfile_MAC_ARM" || export VAGRANT_VAGRANTFILE="Vagrantfile_MAC_INTEL"
vagrant up
vagrant provision
PGPASSWORD=wolfik psql -U marat -h 127.0.0.1 -p 5440 -d testdb
```
