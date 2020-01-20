#!/usr/bin/python
import random
import subprocess
import platform
import urllib2
import json
import zipfile
import os


OS = ""
INSTALL_TOOL = ""
SUPERVISOR_CONFIG_BASE_DIR = ""
GZV_BASE_DIR = os.environ['HOME'] + "/easy_gzv"


SUPERVISOR_CONFIG_MODEL = '''; supervisor config file
[unix_http_server]
file={for_mac}/var/run/supervisor.sock   ; (the path to the socket file)
chmod=0700                       ; sockef file mode (default 0700)

[inet_http_server]
port = 127.0.0.1:9001

[supervisord]
logfile={for_mac}/var/log/supervisord.log ; (main log file;default $CWD/supervisord.log)
pidfile={for_mac}/var/run/supervisord.pid ; (supervisord pidfile;default supervisord.pid)

; the below section must remain in the config file for RPC
; (supervisorctl/web interface) to work, additional interfaces may be
; added by defining them in separate rpcinterface: sections
[rpcinterface:supervisor]
supervisor.rpcinterface_factory = supervisor.rpcinterface:make_main_rpcinterface

[supervisorctl]
serverurl=unix://{for_mac}/var/run/supervisor.sock ; use a unix:// URL  for a unix socket

; The [include] section can just contain the "files" setting.  This
; setting can list multiple files (separated by whitespace or
; newlines).  It can also contain wildcards.  The filenames are
; interpreted as relative to this file.  Included files *cannot*
; include files themselves.

[include]
files = {for_mac}/etc/supervisor.d/*.ini'''

NODE_CONFIG_MODEL = '''[program:{program}]
command=sh miner.sh
directory={work_dir}
stdout_logfile={work_dir}/stdout.log
logfile={work_dir}/monitor_info.log
stdout_logfile_maxbytes=20MB   ; max # logfile bytes b4 rotation (default 50MB)
stdout_logfile_backups=10     ; # of stdout logfile backups (0 means none, default 10)
autostart = true
startsecs = 5
autorestart = true
startretries = 3'''

MONITOR_CONFIG_MODEL = '''[program:monitor]
command=gzv_monitor
directory={work_dir}
stdout_logfile={work_dir}/stdout.log
logfile={work_dir}/monitor_info.log
stdout_logfile_maxbytes=20MB   ; max # logfile bytes b4 rotation (default 50MB)
stdout_logfile_backups=10     ; # of stdout logfile backups (0 means none, default 10)
autostart = true
startsecs = 5
autorestart = true
startretries = 3'''


def init():
    global OS, INSTALL_TOOL, SUPERVISOR_CONFIG_BASE_DIR, SUPERVISOR_CONFIG_MODEL
    res = platform.platform()
    if "ubuntu" in res or "Ubuntu" in res:
        OS = "ubuntu"
        INSTALL_TOOL = "apt"
        SUPERVISOR_CONFIG_BASE_DIR = "/etc/"
        SUPERVISOR_CONFIG_MODEL = SUPERVISOR_CONFIG_MODEL.format(for_mac="")
    elif "centos" in res or "Centos" in res:
        OS = "centos"
        INSTALL_TOOL = "yum"
        SUPERVISOR_CONFIG_BASE_DIR = "/etc/"
        SUPERVISOR_CONFIG_MODEL = SUPERVISOR_CONFIG_MODEL.format(for_mac="")
    elif "Darwin" in res:
        OS = "mac"
        INSTALL_TOOL = "brew"
        SUPERVISOR_CONFIG_BASE_DIR = "/usr/local/etc/"
        SUPERVISOR_CONFIG_MODEL = SUPERVISOR_CONFIG_MODEL.format(for_mac="/usr/local")
    else:
        raise Exception("unsupported system!")


init()


def calculate_version_weight(s):
    weight = 0
    top_weight = 10000000000
    s = s.replace("v", "")
    l = s.split(".")
    try:
        for index, v in enumerate(l):
            weight += top_weight/(100**index)*int(v)
    except Exception:
        pass
    return weight


def remote_gzv_version():
    res = urllib2.urlopen(url="https://update.zvchain.io:8000/request", timeout=20)
    response = json.load(res)
    data = response.get("data").get("data")
    version = data.get("version")
    if OS == "mac":
        url = data.get("update_for_darwin").get("package_url")
    else:
        url = data.get("update_for_linux").get("package_url")
    return version.encode("utf-8"), url.encode("utf-8")


def download_gzv(url):
    subprocess.check_call(["curl", url, "-o", "/tmp/gzv_tmp.zip"])
    zf = zipfile.ZipFile("/tmp/gzv_tmp.zip")
    try:
        zf.extractall("/tmp/gzv_unzip")
    except Exception:
        zf.close()
        raise Exception("download zip error")
    zf.close()


def run(cmd):
    try:
        p = subprocess.Popen(cmd, stdout=subprocess.PIPE, universal_newlines=True)
        p.wait()
        result_lines = p.stdout.readlines()
    except Exception:
        return None
    return result_lines


def gzv_version():
    res = run(["gzv", "version"])
    if res is not None and len(res) > 0:
        return res[0].replace("\n", "").split(" ")[2]
    return None


def supervisor_version():
    res = run(["supervisord", "-version"])
    if res is not None and  len(res) > 0:
        return res[0].replace("\n", "")
    return None


def has_curl():
    res = run(["curl", "--version"])
    if res is not None and len(res) > 0:
        return True
    return False

def install_curl():
    if OS == "centos":
        subprocess.check_call(["yum", "install", "-y", "curl"])
    elif OS == "ubuntu":
        subprocess.check_call(["apt", "install", "-y", "curl"])
    else:
        subprocess.check_call(["brew", "install", "curl"])


def install_supervisor():
    if OS == "centos":
        subprocess.check_call(["yum", "install", "-y", "epel-release"])
        subprocess.check_call(["yum", "install", "-y", "supervisor"])
    elif OS == "ubuntu":
        subprocess.check_call(["apt", "install", "-y", "supervisor"])
    else:
        subprocess.check_call(["brew", "install", "supervisor"])


def install_gzv(url):
    download_gzv(url)
    subprocess.check_call(["cp", "/tmp/gzv_unzip/gzv", "/usr/local/bin"])
    subprocess.check_call(["chmod", "+x", "/usr/local/bin/gzv"])


def make_monitor_env():
    if not os.path.exists("{}/monitor".format(GZV_BASE_DIR)):
        os.mkdir("{}/monitor".format(GZV_BASE_DIR))
    subprocess.check_call(["cp", "gzv_monitor", "/usr/local/bin"])
    subprocess.check_call(["chmod", "+x", "/usr/local/bin/gzv_monitor"])


def gen_password():
    return "".join(random.sample('abcdefghijklmnopqrstuvwxyz0123456789', 10))

def close_firewall():
    subprocess.check_call(["systemctl", "disable", "firewalld.service"])
    subprocess.check_call(["systemctl", "stop", "firewalld.service"])

def add_auto_start():
    with open("/etc/rc.d/rc.local", "r") as f:
        data = f.read()
    if "supervisord" in data:
        return
    with open("/etc/rc.d/rc.local", "a") as f:
        f.write("\n" + "supervisord -c {}supervisord.ini\n".format(SUPERVISOR_CONFIG_BASE_DIR))
        subprocess.check_call(["chmod", "+x", "/etc/rc.d/rc.local"])


def init_supervisor(num):
    if num > 9:
        raise Exception("too many nodes")
    main_config_path = "{}supervisord.ini".format(SUPERVISOR_CONFIG_BASE_DIR)
    if os.path.exists(main_config_path):
        print("rewrite file {}: y or n?".format(main_config_path))
        while True:
            i = raw_input()
            if i == "y":
                break
            elif i == "n":
                raise Exception("user stop.")
            else:
                print("rewrite file {}: y or n?".format(main_config_path))
    with open(main_config_path, "w") as f:
        f.write(SUPERVISOR_CONFIG_MODEL)
    configs_path = "{}supervisor.d".format(SUPERVISOR_CONFIG_BASE_DIR)
    if not os.path.exists(configs_path):
        print("generate {} config file dir".format(configs_path))
        os.mkdir(configs_path)
    for index in range(num):
        program_name = "gzv"+str(index+1)
        work_dir = "{}/gzv_run{}".format(GZV_BASE_DIR, index+1)
        config_path = "{}/{}.ini".format(configs_path, program_name)
        if not os.path.exists(work_dir):
            print("generate dir {}".format(work_dir))
            os.mkdir(work_dir)
        else:
            print("skip :exist {} dir".format(work_dir))
        if not os.path.exists("{}/miner.sh".format(work_dir)):
            with open("{}/miner.sh".format(work_dir), "w") as f:
                f.write(
                    '''ulimit -HSn 65535 \ngzv miner --rpc 3 --host 127.0.0.1 --port 810{index} --chainid 1 --createaccount --password {password}'''.format(
                        password=gen_password(), index=index + 1))
            subprocess.check_call(["chmod", "+x", "{}/miner.sh".format(work_dir)])
        if not os.path.exists(config_path):
            with open(config_path, "w") as f:
                f.write(NODE_CONFIG_MODEL.format(program=program_name, work_dir=work_dir))
        else:
            print("skip :exist {} config file".format(config_path))
    # write monitor ini
    monitor_config_path = "{}/monitor.ini".format(configs_path)
    if not os.path.exists(monitor_config_path):
        with open(monitor_config_path, "w") as f:
            f.write(MONITOR_CONFIG_MODEL.format(work_dir="{}/monitor".format(GZV_BASE_DIR)))
    else:
        print("skip :exist {} config file".format(monitor_config_path))
    subprocess.check_call(["supervisord", "-c", main_config_path])







if __name__ == '__main__':
    if not os.path.exists(GZV_BASE_DIR):
        os.mkdir(GZV_BASE_DIR)
    if not has_curl():
        install_curl()
    su_version = supervisor_version()
    if su_version is None:
        print("start install supervisor")
        install_supervisor()
    g_version = gzv_version()
    version, url = remote_gzv_version()
    if g_version is None:
        print("start install gzv")
        install_gzv(url)
    else:
        local_weight = calculate_version_weight(g_version)
        remote_weight = calculate_version_weight(version)
        if remote_weight > local_weight:
            print("update gzv, y or n?")
            while True:
                i = raw_input()
                if i == "y":
                    install_gzv(url)
                elif i == "n":
                    break
                else:
                    print("y or n?")
    make_monitor_env()
    gzv_num = os.environ.get("GZV_NUM", "2")
    init_supervisor(int(gzv_num))
    close_firewall()
    add_auto_start()





