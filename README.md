# FileAutoBackup

给没有自动备份的sb palworld服务端写的自动文件复制器

### Notice

Unix平台可能会因为路径符原因导致不生效，没测试过，但是大概应该没问题

`files` 字段需要注意程序无法将字符串反序列化成字符串切片，要按数组的形式写

```yaml
palworld:
  dir: "D:\\Game\\SteamCMD\\steamapps\\common\\PalServer\\Pal\\Saved\\SaveGames\\0\\259201A849EBE1C85131B0ADF18F8F7A"
#  files: #none for copying the whole folder
#    - "Level.sav"
  copyTo: "D:\\Game\\SteamCMD\\steamapps\\common\\PalServer\\Pal\\Saved\\SaveGames\\0\\backup"
  compression: true
  minimumInterval: "5m" #unit: "s", "m", "h"    , not less than 1s
  timeToDeleteOld: "168h" #Not implemented
  countTodeleteOld: 0 #Not implemented
```