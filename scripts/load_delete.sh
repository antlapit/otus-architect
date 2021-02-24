while TRUE; do ab -n 50 -c 5 -m DELETE -H'Host: arch.homework' http://arch.homework/otusapp/alapitskii/user/1 ; sleep 3; done
