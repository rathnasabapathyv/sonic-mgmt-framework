#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
###########################################################################

import syslog as log
import sys
import cli_client as cc
from rpipe_utils import pipestr
from scripts.render_cli import show_cli_output
import urllib3
urllib3.disable_warnings()
import subprocess
import re

#Invalid chars
blocked_chars = frozenset(['&', ';', '<', '>', '|', '`', '\''])
api = cc.ApiClient()

def is_std_mode():
    path = cc.Path('/restconf/data/sonic-device-metadata:sonic-device-metadata/DEVICE_METADATA/DEVICE_METADATA_LIST={name}/intf_naming_mode', name="localhost")
    response = api.get(path)
    if response is None:
        return False

    if response.ok():
        response = response.content
        if response is None:
            return False

    response = response.get('sonic-device-metadata:intf_naming_mode')
    if response is None:
        return False

    if "standard" in response.lower():
        return True
    else:
        return False


def get_alias(interface):
    path = cc.Path('/restconf/data/sonic-port:sonic-port/PORT_TABLE/PORT_TABLE_LIST={name}/alias', name=interface)
    response = api.get(path)

    if response is None:
        return None

    if response.ok():
        response = response.content
        if response is None:
            return None

        interface = response.get('sonic-port:alias')
        return interface

def print_and_log(msg):
    print "% Error: ", msg
    log.syslog(log.LOG_ERR, msg)

def run_vrf(args, ping, vrfName):
    if len(args) == 0:
        print_and_log("The command is not complete.")
        return
    try:
        args = re.sub(r"(PortChannel|Ethernet|Management|Loopback|Vlan)(\s+)(\d+)", "\g<1>\g<3>", args)
        if ping == "ping6":
            if vrfName.lower() == 'mgmt':
                cmd = "sudo cgexec -g l3mdev:" + vrfName + " ping -6 " + args
            else:
                cmd = "ping -I " + vrfName + " " + args
        else:
            if vrfName.lower() == 'mgmt':
                cmd = "sudo cgexec -g l3mdev:" + vrfName + " ping -6 " + args
            else:
                cmd = "ping -I " + vrfName + " " + args

        cmd = re.sub('-I\s*Management', '-I eth', cmd)
        cmdList = cmd.split(' ')
        subprocess.call(cmdList, shell=False)

    except KeyboardInterrupt:
        # May be triggered when Ctrl + C is used to stop script execution
        return
    except Exception as e:
        print_and_log("Internal error")
        log.syslog(log.LOG_ERR, str(e))
        return

def run(args, ping):
    if len(args) == 0:
        print_and_log("The command is not complete.")
        return
    try:
        args = re.sub(r"(PortChannel|Ethernet|Management|Loopback|Vlan)(\s+)(\d+)", "\g<1>\g<3>", args)
        if ping == "ping6":
            cmd = "ping -6 " + args
        else:
            cmd = "ping " + args

        cmd = re.sub('-I\s*Management', '-I eth', cmd)
        cmdList = cmd.split(' ')
        subprocess.call(cmdList, shell=False)

    except KeyboardInterrupt:
        # May be triggered when Ctrl + C is used to stop script execution
        return
    except Exception as e:
        print_and_log("Internal error")
        log.syslog(log.LOG_ERR, str(e))
        return

def validate_input(args):
    if(set(args) & blocked_chars):
        print_and_log("Invalid argument")
        return False, args

    result = re.search('(-I\s*)(Eth\d+/\d+/*\d*)', args, re.IGNORECASE)
    if (result is not None) and (is_std_mode()):
        interface = result.group(2)
        alias = get_alias(interface)
        if alias is None:
            return False, args
        args = re.sub('(-I\s*)(Eth\d+/\d+/*\d*)', '\g<1>' + alias, args, re.IGNORECASE)

    if ("fe80:" in args.lower()
        or "ff01:" in args.lower()
        or "ff02:" in args.lower()):
        if "vrf" in args:
            print_and_log("VRF name is not allowed for IPv6 addresses with link-local scope")
            return False, args
        if " -I" not in args:
            print_and_log("Interface name was missing")
            return False, args
    return True, args

if __name__ == '__main__':
    pipestr().write(sys.argv)
    isVrf = False

    if len(sys.argv) > 2 and sys.argv[2] == "vrf":
        args = " ".join(sys.argv[4:])
        isVrf = True
    else:
        args = " ".join(sys.argv[2:])

    status, args = validate_input(args)
    if status is False:
        sys.exit(1)

    if isVrf:
        run_vrf(args, sys.argv[1], sys.argv[3])
    else:
        run(args, sys.argv[1])
