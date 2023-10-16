# Puprose
grelay is simple tool to forward tcp traffic through some transitional host that has access to some network where traffic should be forwarded.

## When it may be helpful?
Use cases when application on target host doesn't have access to specificc network e.g VPN/subnet/tunnels on the other hand a transitional machine does.
Relay is launched on transitional host and application on target host could use ip address of transitional host as destination address.
Keep it in mind, you have to know what ports is used by your application. Assumed that no random ports are opened during application work.

## Usage example
Let's forwards all traffic came on transitional host 192.168.0.42 on ports 1072,2042 to some remote host located by address 10.0.0.72
```Shell
grelay -l 192.168.0.42 -r 10.0.0.72 -p 1072,2042
```

### Command line arguments
* -l `some ipv4 address where incoming traffic is come i.e. one of addresses on transitional host which is visible for target host/application`
* -r `remote address somethere in target vpn/subnet/tunnel`
* -p `comma separated port list to be forwarded`

## Q&A
* Q: Why just not configure VPN/routing on router?
  A: It may take a lot of time.
* Q: There is no local address autodetection. Why?
  A: There could be numerous of networks on transit host, that is, using some randomly selected address doesn't have sense.
