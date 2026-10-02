# 星语桥后端冒烟测试脚本
# 前置：服务已启动（go run .），WECHAT_MOCK=true，AI_PROVIDER=mock 或已配置 DeepSeek
param(
    [string]$BaseUrl = "http://127.0.0.1:8080/api/v1"
)

$ErrorActionPreference = "Stop"

function Invoke-Api {
    param(
        [string]$Method,
        [string]$Path,
        $Body,
        [string]$Token
    )
    $headers = @{}
    if ($Token) { $headers["Authorization"] = "Bearer $Token" }
    $params = @{
        Uri     = "$BaseUrl$Path"
        Method  = $Method
        Headers = $headers
        TimeoutSec = 30
    }
    if ($null -ne $Body) {
        $json = $Body | ConvertTo-Json -Compress
        # 用 UTF-8 字节发送，避免中文乱码
        $params["Body"] = [System.Text.Encoding]::UTF8.GetBytes($json)
        $params["ContentType"] = "application/json; charset=utf-8"
    }
    return Invoke-RestMethod @params
}

Write-Host "== 1. 登录 / 同意 / 画像 ==" -ForegroundColor Cyan
$login = Invoke-Api -Method Post -Path "/auth/login" -Body @{ code = "smoke-" + (Get-Random) }
$token = $login.data.token
Write-Host "登录成功 user_id=$($login.data.user.id) need_consent=$($login.data.need_consent)"

$consent = Invoke-Api -Method Post -Path "/user/consent" -Body @{ consent = 1 } -Token $token
Write-Host "提交同意后 need_consent=$($consent.data.need_consent)"

$child = Invoke-Api -Method Put -Path "/user/child" -Token $token -Body @{
    name              = "小明"
    age               = 6
    language_level    = "短句"
    interests         = "积木,火车"
    prompt_preference = "图片优先"
    wait_seconds      = 5
    stop_signal       = "说不要了"
}
Write-Host "儿童画像已保存 id=$($child.data.child.id) name=$($child.data.child.name)"

Write-Host "`n== 2. 情境列表 ==" -ForegroundColor Cyan
$scenarios = (Invoke-Api -Method Get -Path "/scenarios" -Token $token).data.scenarios
$scenarios | ForEach-Object { Write-Host " - $($_.id) : $($_.name)" }

Write-Host "`n== 3. 正常闭环（加入搭积木）==" -ForegroundColor Cyan
$s = (Invoke-Api -Method Post -Path "/training/sessions" -Token $token -Body @{ scenario_id = "build_blocks" }).data
Write-Host "开始训练 session=$($s.session_id) 第 $($s.round) 轮：$($s.question)"
Write-Host "可选词块：$($s.word_blocks -join ' / ')"

foreach ($answer in @("我想和你们一起玩积木", "我想搭一个城堡", "我想再来一次")) {
    $r = (Invoke-Api -Method Post -Path "/training/sessions/$($s.session_id)/answers" -Token $token -Body @{ text = $answer }).data
    Write-Host "[儿童] $answer"
    Write-Host "   判断=$($r.validity) 动作=$($r.next_action) 反馈=$($r.feedback)"
    if ($r.next_action -eq "next_turn") { Write-Host "   下一轮：$($r.question)" }
}
Write-Host "训练总结：$($r.summary)"
Write-Host "任务卡：$($r.task_card.name) / 目标句：$($r.task_card.target_sentence)"

Write-Host "`n== 4. 三级重试降级（户外休息）==" -ForegroundColor Cyan
$s2 = (Invoke-Api -Method Post -Path "/training/sessions" -Token $token -Body @{ scenario_id = "outdoor_needs" }).data
foreach ($answer in @("今天天气不错", "我想去公园滑梯", "我想")) {
    $r = (Invoke-Api -Method Post -Path "/training/sessions/$($s2.session_id)/answers" -Token $token -Body @{ text = $answer }).data
    Write-Host "[儿童] $answer -> 判断=$($r.validity) 动作=$($r.next_action) 重试=$($r.retry_count)"
    if ($r.target_sentence) { Write-Host "   目标句提示：$($r.target_sentence)" }
}

Write-Host "`n== 5. 高风险内容安全中止 ==" -ForegroundColor Cyan
$s3 = (Invoke-Api -Method Post -Path "/training/sessions" -Token $token -Body @{ scenario_id = "build_blocks" }).data
$risk = (Invoke-Api -Method Post -Path "/training/sessions/$($s3.session_id)/answers" -Token $token -Body @{ text = "我想打人" }).data
Write-Host "risk_flag=$($risk.risk_flag) status=$($risk.status)"
Write-Host "儿童端提示：$($risk.feedback)"
Write-Host "家长端提示：$($risk.parent_tip)"

Write-Host "`n== 6. 任务卡与回填 ==" -ForegroundColor Cyan
$cards = (Invoke-Api -Method Get -Path "/task-cards" -Token $token).data.task_cards
Write-Host "任务卡数量：$(@($cards).Count)"
$card = @($cards)[0]
$back = (Invoke-Api -Method Post -Path "/task-cards/$($card.id)/backfill" -Token $token -Body @{ feedback = 1; note = "孩子独立完成" }).data
Write-Host "回填结果：status=$($back.status) feedback=$($back.feedback) note=$($back.note)"

Write-Host "`n全部流程测试通过。" -ForegroundColor Green
