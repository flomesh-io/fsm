# dnsmasq

- Setup **dnsmasq** to resolve *.localhost domain to 127.0.0.1
  * Install: `brew install dnsmasq`
  * Create config directory: `mkdir -pv $(brew --prefix)/etc/`
  * Setup *.localhost domain: `echo 'address=/.localhost/127.0.0.1' >> $(brew --prefix)/etc/dnsmasq.conf`
  * Change DNS port: `echo 'port=53' >> $(brew --prefix)/etc/dnsmasq.conf`
  * Autostart dnsmasq - now and after reboot: `sudo brew services start dnsmasq`
  * Create resolver directory: `sudo mkdir -v /etc/resolver`
  * Add your nameserver to resolvers: `sudo bash -c 'echo "nameserver 127.0.0.1" > /etc/resolver/localhost'`