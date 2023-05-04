
export OLDLOGDIR=$INTERMAIL/oldlogs
export mx_log=$INTERMAIL/log
export mxos_logs=$INTERMAIL/mxos/logs
export mxosserver_logs=$INTERMAIL/mxos/server/logs
export mira_logs=$INTERMAIL/mira/usr/store/logs
for d in ${mx_log} ${mxos_logs} ${mxosserver_logs} ${micra_logs}
do ./oldlogzipper ${d}
bash oldlogzipmv.sh ${d} ${OLDLOGDIR}
done
