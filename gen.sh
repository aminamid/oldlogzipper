for x in mta mss imapserv popserv imdirserv imdircacheserv immgrserv imconfserv imqueueueserv smtp launchd dp counters
 do for y in log trace stat
  do echo "(${x})\\..*\\.20\\d{10}[-+]\\d{4}.${y}\$"
 done
done
echo "(mxos\.log)(\.\d+){0,1}$"
echo "(mxos\.stats)(\.\d+){0,1}$"
echo "(localhost_access_log).[-\d]+\.txt$"
echo "(localhost).[-\d]+\.log$"
echo "(gc)\.log\.*"
echo "(catalina)\.(out|[-\d]+\.log)$"
echo "(manager)\.[-\d]+\.log$"
echo "(host-manager)\.[-\d]+\.log$"
