# Copyright 2016 祝景法(Bruce)@haimi.com. www.haimi.com All rights reserved.
from cluster.info import Info


def GetServerInfo(info):
    return ServerInfo(info.Event, info.Service, info.Backend, info.Iplist, info.Localip, info.Masterip, info.Qurorum)


class ServerInfo(Info):
    'Cluster Zookeeper Boot info'

    ServerIDAdd = 1

    ClientPort = 2181

    def __init__(self, event, service, backend, iplist, localip, masterip, qurorum):
        Info.__init__(self, event, service, backend, iplist, localip, masterip, qurorum)

    def GetServerList(self):
        list = []

        IPList = self.GetIPListArray()
        for serverid in range(len(IPList)):
            # start from 1
            list.append("server." + str(serverid + self.ServerIDAdd) + "=" + IPList[serverid] + ":2888:3888")

        return "\n".join(list)

    def GetMyID(self):
        IPList = self.GetIPListArray()
        for serverid in range(len(IPList)):
            if IPList[serverid] == self.Localip:
                return serverid + self.ServerIDAdd

        return 0

    def GetLocalClientURL(self):
        return self.GetServiceUrl(self.ClientPort)

    def GetAllClientURL(self):
        list = []

        IPList = self.GetIPListArray()
        for ip in IPList:
            # start from 1
            list.append(ip + ":" + str(self.ClientPort))

        return list


if __name__ == "__main__":
    info = ServerInfo("OnStart", "etcd", "etcd", "192.168.1.1, 192.168.1.2", "192.168.1.2", "192.168.1.2", "3")
    print(info.Backend, info.Iplist, info.Localip)
    print(info.GetIPListArray())
    print(info.CheckLocalIp())
    if info.GetServiceUrl(8500) != "192.168.1.2:8500":
        print("Error: info.GetServiceUrl(8500)!=192.168.1.2:8500")

    if info.GetNodename() != "Etcd-192.168.1.2":
        print("Error: info.GetNodename() != Etcd-192.168.1.2")

    if info.GetMyID() != 2:
        print("info.GetMyID() found error", info.GetMyID())

    info = ServerInfo("OnStart", "etcd", "etcd", "192.168.1.1, 192.168.1.2", "192.168.1.5", "192.168.1.2", "3")
    print(info.Backend, info.Iplist, info.Localip)
    print(info.GetIPListArray())
    print(info.CheckLocalIp())

    print(info.GetServerList())
    if info.GetServerList() != "server.1=192.168.1.1:2888:3888\nserver.2=192.168.1.2:2888:3888":
        print("info.GetServerList() found error")

    if info.GetMyID() != 0:
        print("info.GetMyID() found error", info.GetMyID())

    # Create from info
    print("Test Create from info...")
    infoInst = Info("OnStart", "etcd", "etcd", "192.168.1.1, 192.168.1.2", "192.168.1.2", "192.168.1.2", "3")
    info = GetServerInfo(infoInst)
    print(info.Backend, info.Iplist, info.Localip)
    print(info.GetIPListArray())
    print(info.CheckLocalIp())
    print(info.GetServerList())
    if info.GetServiceUrl(8500) != "192.168.1.2:8500":
        print("Error: info.GetServiceUrl(8500)!=192.168.1.2:8500")

    if info.GetServiceUrl(info.ClientPort) != "192.168.1.2:2181":
        print("Error: info.GetServiceUrl(2181)!=192.168.1.2:2181")

    if info.GetLocalClientURL() != "192.168.1.2:2181":
        print("Error: info.GetLocalClientURL()!=192.168.1.2:2181")
    if info.GetNodename() != "Etcd-192.168.1.2":
        print("Error: info.GetNodename() != Etcd-192.168.1.2")

    if info.GetMyID() != 2:
        print("info.GetMyID() found error", info.GetMyID())

    print("Client urls:", info.GetAllClientURL())
    if (info.GetAllClientURL() != ['192.168.1.1:2181', '192.168.1.2:2181']):
        print("Client urls: info.GetAllClientURL()", info.GetAllClientURL())
