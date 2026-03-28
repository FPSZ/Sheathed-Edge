set -euo pipefail
pkill gateway || true
cd /mnt/d/AI/Local/Agent/gateway-go
source /etc/profile.d/sheathed-edge-env.sh
cat > /tmp/toolcall.json <<'EOF'
{"model":"awdp-r1-70b","messages":[{"role":"user","content":"请先使用 retrieval 工具搜索本地知识库中与 AWDP 有关的内容，再用中文总结 3 条要点。必须优先检索，不要直接凭空回答。"}],"stream":false}
EOF
( GOTRACEBACK=all go run ./cmd/gateway -config /mnt/d/AI/Local/Agent/gateway.config.json > /tmp/gateway_fg.log 2>&1 ) &
gw=$!
sleep 5
curl -sS -D /tmp/gateway_resp.h -o /tmp/gateway_resp.b http://127.0.0.1:8090/v1/chat/completions -H "Content-Type: application/json" --data-binary @/tmp/toolcall.json || true
sleep 2
echo ====HEADERS====
cat /tmp/gateway_resp.h 2>/dev/null || true
echo ====BODY====
cat /tmp/gateway_resp.b 2>/dev/null || true
echo
echo ====PROC====
ps -p $gw -o pid=,stat=,etime=,cmd= || true
echo ====LOG====
tail -n 200 /tmp/gateway_fg.log 2>/dev/null || true
kill $gw 2>/dev/null || true