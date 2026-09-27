"use strict";

const I18N = {
  en: {
    good: "GOOD", caution: "CAUTION", bad: "BAD", unknown: "UNKNOWN", ok: "OK",
    health: "Health", temperature: "Temperature", firmware: "Firmware", serial: "Serial Number",
    interface: "Interface", power_on: "Power On Hours", power_on_count: "Power On Count",
    rotation: "Rotation Rate", rpm: "RPM", ssd: "SSD", device: "Device", smart: "SMART",
    life: "Life", status_reasons: "Status",
    rescan: "Rescan", updated: "Updated", theme: "Theme",
    theme_classic: "Classic", theme_dark: "Dark", theme_follow: "Follow fnOS", lang: "Language",
    no_disks: "No disks found", no_attrs: "No SMART attributes", hours: "hours",
    selftest: "Self-Test", aam_apm: "AAM/APM", alarms: "Alarms", graph: "Graph",
    report: "Report", raw_json: "Raw JSON", settings: "Settings",
    test_type: "Test type", short: "Short", long: "Long", conveyance: "Conveyance",
    start: "Start", abort: "Abort", last_test: "Last test", test_output: "Result",
    close: "Close", apply: "Apply", current: "Current", value: "Value", off: "Disable", set: "Set",
    no_alarms: "No alarms", time: "Time", disk: "Disk", kind: "Kind", from: "From", to: "To",
    health_change: "Health", temperature_change: "Temperature", dismiss: "Dismiss",
    metric: "Metric", points: "Points", all: "All", select_disks: "Disks", draw: "Draw",
    col_id: "ID", col_attr: "Attribute", col_cur: "Current", col_worst: "Worst",
    col_thr: "Threshold", col_raw: "Raw", col_status: "Status",
    server: "Server", client: "Display", interval_seconds: "Interval (seconds)",
    alarm_temp: "Alarm temp", threshold: "Threshold", defaults: "Defaults", per_disk: "Per disk",
    unit: "Temperature unit", celsius: "Celsius", fahrenheit: "Fahrenheit",
    raw_format: "Raw value format", hex: "16 HEX", dec: "10 DEC", byte2: "10 DEC 2byte", byte1: "10 DEC 1byte",
    hide_serial: "Hide serial number", saved: "Saved", inherit: "inherit",
    m_temperature: "Temperature", m_life: "Life", m_power_on_hours: "Power On Hours",
    m_power_on_count: "Power On Count", m_reallocated: "Reallocated Sectors",
    m_realloc_events: "Realloc. Events", m_pending: "Pending Sectors",
    m_uncorrectable: "Uncorrectable", m_host_reads: "Host Reads (GB)", m_host_writes: "Host Writes (GB)",
    attr_1: "Raw Read Error Rate", attr_3: "Spin Up Time", attr_4: "Start/Stop Count",
    attr_5: "Reallocated Sectors Count", attr_7: "Seek Error Rate", attr_9: "Power On Hours",
    attr_10: "Spin Retry Count", attr_12: "Power Cycle Count", attr_13: "Read Soft Error Rate",
    attr_183: "Runtime Bad Block", attr_184: "End-to-End Error", attr_187: "Reported Uncorrectable",
    attr_188: "Command Timeout", attr_190: "Airflow Temperature", attr_192: "Power-off Retract",
    attr_193: "Load Cycle Count", attr_194: "Temperature", attr_196: "Reallocation Event Count",
    attr_197: "Current Pending Sectors", attr_198: "Offline Uncorrectable", attr_199: "UDMA CRC Errors",
    attr_200: "Multi-Zone Error Rate", attr_240: "Head Flying Hours", attr_241: "Total LBAs Written",
    attr_242: "Total LBAs Read"
  },
  "zh-CN": {
    good: "良好", caution: "注意", bad: "损坏", unknown: "未知", ok: "正常",
    health: "健康状态", temperature: "温度", firmware: "固件", serial: "序列号",
    interface: "接口", power_on: "通电时间", power_on_count: "通电次数",
    rotation: "转速", rpm: "RPM", ssd: "固态", device: "设备", smart: "SMART",
    life: "寿命", status_reasons: "状态",
    rescan: "重新扫描", updated: "更新于", theme: "主题",
    theme_classic: "经典", theme_dark: "深色", theme_follow: "跟随 fnOS", lang: "语言",
    no_disks: "未发现硬盘", no_attrs: "无 SMART 属性", hours: "小时",
    selftest: "自检", aam_apm: "AAM/APM", alarms: "报警", graph: "图表",
    report: "报告", raw_json: "原始 JSON", settings: "设置",
    test_type: "检测类型", short: "短检测", long: "长检测", conveyance: "传送检测",
    start: "开始", abort: "中止", last_test: "上次检测", test_output: "结果",
    close: "关闭", apply: "应用", current: "当前", value: "值", off: "关闭", set: "设置",
    no_alarms: "无报警", time: "时间", disk: "硬盘", kind: "类型", from: "从", to: "到",
    health_change: "健康变化", temperature_change: "温度变化", dismiss: "忽略",
    metric: "指标", points: "点数", all: "全部", select_disks: "硬盘", draw: "绘制",
    col_id: "ID", col_attr: "属性", col_cur: "当前", col_worst: "最差",
    col_thr: "阈值", col_raw: "原始值", col_status: "状态",
    server: "服务端", client: "显示", interval_seconds: "采集间隔（秒）",
    alarm_temp: "报警温度", threshold: "阈值", defaults: "默认值", per_disk: "每块硬盘",
    unit: "温度单位", celsius: "摄氏度", fahrenheit: "华氏度",
    raw_format: "原始值格式", hex: "16 进制", dec: "10 进制", byte2: "10 进制 2字节", byte1: "10 进制 1字节",
    hide_serial: "隐藏序列号", saved: "已保存", inherit: "继承",
    m_temperature: "温度", m_life: "寿命", m_power_on_hours: "通电时间",
    m_power_on_count: "通电次数", m_reallocated: "重映射扇区", m_realloc_events: "重映射事件",
    m_pending: "待映射扇区", m_uncorrectable: "不可纠正扇区",
    m_host_reads: "主机读取 (GB)", m_host_writes: "主机写入 (GB)",
    attr_1: "底层读取错误率", attr_3: "旋转时间", attr_4: "启动次数", attr_5: "重映射扇区计数",
    attr_7: "寻道错误率", attr_9: "通电时间", attr_10: "旋转重试计数", attr_12: "断电启动次数",
    attr_13: "读取软错误率", attr_183: "运行中降级事件", attr_184: "端到端错误",
    attr_187: "报告的不可纠正错误", attr_188: "命令超时", attr_190: "气流温度",
    attr_192: "断电退刀次数", attr_193: "磁头加载次数", attr_194: "温度",
    attr_196: "重映射事件计数", attr_197: "当前待映射扇区", attr_198: "离线不可纠正扇区",
    attr_199: "UDMA CRC 错误", attr_200: "多区域错误", attr_240: "磁头飞行时间",
    attr_241: "写入 LBA 总数", attr_242: "读取 LBA 总数"
  },
  "zh-TW": {
    good: "良好", caution: "注意", bad: "損壞", unknown: "未知", ok: "正常",
    health: "健康狀態", temperature: "溫度", firmware: "韌體", serial: "序號",
    interface: "介面", power_on: "通電時間", power_on_count: "通電次數",
    rotation: "轉速", rpm: "RPM", ssd: "固態", device: "裝置", smart: "SMART",
    life: "壽命", status_reasons: "狀態",
    rescan: "重新掃描", updated: "更新於", theme: "主題",
    theme_classic: "經典", theme_dark: "深色", theme_follow: "跟隨 fnOS", lang: "語言",
    no_disks: "未發現硬碟", no_attrs: "無 SMART 屬性", hours: "小時",
    selftest: "自我檢測", aam_apm: "AAM/APM", alarms: "警報", graph: "圖表",
    report: "報告", raw_json: "原始 JSON", settings: "設定",
    test_type: "檢測類型", short: "短檢測", long: "長檢測", conveyance: "搬運檢測",
    start: "開始", abort: "中止", last_test: "上次檢測", test_output: "結果",
    close: "關閉", apply: "套用", current: "目前", value: "值", off: "關閉", set: "設定",
    no_alarms: "無警報", time: "時間", disk: "硬碟", kind: "類型", from: "從", to: "到",
    health_change: "健康變化", temperature_change: "溫度變化", dismiss: "忽略",
    metric: "指標", points: "點數", all: "全部", select_disks: "硬碟", draw: "繪製",
    col_id: "ID", col_attr: "屬性", col_cur: "目前", col_worst: "最差",
    col_thr: "閾值", col_raw: "原始值", col_status: "狀態",
    server: "伺服端", client: "顯示", interval_seconds: "採集間隔（秒）",
    alarm_temp: "警報溫度", threshold: "閾值", defaults: "預設值", per_disk: "每顆硬碟",
    unit: "溫度單位", celsius: "攝氏", fahrenheit: "華氏",
    raw_format: "原始值格式", hex: "16 進位", dec: "10 進位", byte2: "10 進位 2位元組", byte1: "10 進位 1位元組",
    hide_serial: "隱藏序號", saved: "已儲存", inherit: "繼承",
    m_temperature: "溫度", m_life: "壽命", m_power_on_hours: "通電時間",
    m_power_on_count: "通電次數", m_reallocated: "重映射磁區", m_realloc_events: "重映射事件",
    m_pending: "待映射磁區", m_uncorrectable: "不可更正磁區",
    m_host_reads: "主機讀取 (GB)", m_host_writes: "主機寫入 (GB)",
    attr_1: "底層讀取錯誤率", attr_3: "旋轉時間", attr_4: "啟動次數", attr_5: "重映射磁區計數",
    attr_7: "尋道錯誤率", attr_9: "通電時間", attr_10: "旋轉重試計數", attr_12: "斷電啟動次數",
    attr_13: "讀取軟錯誤率", attr_183: "執行中降級事件", attr_184: "端到端錯誤",
    attr_187: "回報的不可更正錯誤", attr_188: "命令逾時", attr_190: "氣流溫度",
    attr_192: "斷電退刀次數", attr_193: "磁頭載入次數", attr_194: "溫度",
    attr_196: "重映射事件計數", attr_197: "目前待映射磁區", attr_198: "離線不可更正磁區",
    attr_199: "UDMA CRC 錯誤", attr_200: "多區域錯誤", attr_240: "磁頭飛行時間",
    attr_241: "寫入 LBA 總數", attr_242: "讀取 LBA 總數"
  },
  ja: {
    good: "良好", caution: "注意", bad: "異常", unknown: "不明", ok: "正常",
    health: "健康状態", temperature: "温度", firmware: "ファームウェア", serial: "シリアル番号",
    interface: "インターフェース", power_on: "通電時間", power_on_count: "通電回数",
    rotation: "回転数", rpm: "RPM", ssd: "SSD", device: "デバイス", smart: "S.M.A.R.T.",
    life: "寿命", status_reasons: "状態",
    rescan: "再スキャン", updated: "更新", theme: "テーマ",
    theme_classic: "クラシック", theme_dark: "ダーク", theme_follow: "fnOS に従う", lang: "言語",
    no_disks: "ディスクが見つかりません", no_attrs: "SMART 属性なし", hours: "時間",
    selftest: "セルフテスト", aam_apm: "AAM/APM", alarms: "アラーム", graph: "グラフ",
    report: "レポート", raw_json: "生 JSON", settings: "設定",
    test_type: "テストの種類", short: "ショート", long: "ロング", conveyance: "コンベヤ",
    start: "開始", abort: "中止", last_test: "前回のテスト", test_output: "結果",
    close: "閉じる", apply: "適用", current: "現在値", value: "値", off: "オフ", set: "設定",
    no_alarms: "アラームなし", time: "時刻", disk: "ディスク", kind: "種類", from: "前", to: "後",
    health_change: "健康状態", temperature_change: "温度", dismiss: "無視",
    metric: "メトリック", points: "データ点数", all: "すべて", select_disks: "ディスク", draw: "描画",
    col_id: "ID", col_attr: "属性", col_cur: "現在", col_worst: "最悪",
    col_thr: "しきい値", col_raw: "生値", col_status: "状態",
    server: "サーバー", client: "表示", interval_seconds: "収集間隔（秒）",
    alarm_temp: "アラーム温度", threshold: "しきい値", defaults: "デフォルト", per_disk: "ディスクごと",
    unit: "温度単位", celsius: "摂氏", fahrenheit: "華氏",
    raw_format: "生値形式", hex: "16 進", dec: "10 進", byte2: "10 進 2バイト", byte1: "10 進 1バイト",
    hide_serial: "シリアル非表示", saved: "保存しました", inherit: "継承",
    m_temperature: "温度", m_life: "寿命", m_power_on_hours: "通電時間",
    m_power_on_count: "通電回数", m_reallocated: "代替処理済みセクタ", m_realloc_events: "代替処理イベント",
    m_pending: "保留中セクタ", m_uncorrectable: "訂正不能セクタ",
    m_host_reads: "ホスト読み込み (GB)", m_host_writes: "ホスト書き込み (GB)",
    attr_1: "生のリードエラー率", attr_3: "スピンアップ時間", attr_4: "起動回数", attr_5: "代替処理済みセクタ数",
    attr_7: "シークエラー率", attr_9: "使用時間", attr_10: "スピン再試行回数", attr_12: "電源投入回数",
    attr_13: "リードソフトエラー率", attr_183: "ランタイムバッドブロック", attr_184: "エンドツーエンドエラー",
    attr_187: "報告された訂正不能エラー", attr_188: "コマンドタイムアウト", attr_190: "エアフロー温度",
    attr_192: "電源断時退避回数", attr_193: "ロードサイクル回数", attr_194: "温度",
    attr_196: "代替処理イベント数", attr_197: "現在保留中のセクタ数", attr_198: "オフライン訂正不能セクタ数",
    attr_199: "UDMA CRC エラー", attr_200: "マルチゾーンエラー率", attr_240: "ヘッド飛行時間",
    attr_241: "書き込み LBA 総数", attr_242: "読み込み LBA 総数"
  }
};

let LANG = (navigator.language || "en").toLowerCase();
if (LANG.startsWith("zh")) {
  LANG = LANG.includes("tw") || LANG.includes("hk") ? "zh-TW" : "zh-CN";
} else if (!LANG.startsWith("ja")) {
  LANG = "en";
}

function setLang(lang) {
  if (I18N[lang]) {
    LANG = lang;
  }
}

function t(key) {
  const m = I18N[LANG] || {};
  return m[key] || I18N.en[key] || key;
}

function attrName(id, fallback) {
  const m = I18N[LANG] || {};
  const name = fallback || "";
  if (m["attr_" + id]) {
    return m["attr_" + id];
  }
  if (LANG === "en") {
    return name.replace(/_/g, " ") || "ID " + id;
  }
  return name.replace(/_/g, " ") || "ID " + id;
}
