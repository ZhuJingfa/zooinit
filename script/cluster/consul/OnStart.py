# Copyright 2016 祝景法(Bruce)@haimi.com. www.haimi.com All rights reserved.
import subprocess
import sys
import io
import time
from cluster.info import Info
from subcall import runcmd


def run(info):
    if (not isinstance(info, Info)):
        print(__name__ + "::run() info is not instance Info, please check")
        sys.exit(1)

    args = ["consul", "agent",
            "-node=" + info.GetNodename(),
            "-data-dir=/data/consul",
            "-bind=" + info.Localip,
            "-client=" + info.Localip]

    # All need server mode to boot up.
    args.append("-server")
    args.append("-bootstrap-expect")
    args.append(info.Qurorum)

    if (info.Localip != info.Masterip):  # slave mode
        args.append("-join=" + info.Masterip)
        # Consul need to wait
        sec = 2
        print("This is a slave node, wait master " + str(sec) + " sec")
        time.sleep(sec)

    runcmd.runWithStdoutSync(args)


# ImportError: No module named cluster.utils
# see readme.md set PYTHONPATH
if __name__ == "__main__":
    run('test')
