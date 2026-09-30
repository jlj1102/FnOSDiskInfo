"use strict";

const I18N = {
  en: {
    good: "GOOD", caution: "CAUTION", bad: "BAD", unknown: "UNKNOWN", ok: "OK",
    health: "Health", temperature: "Temperature", firmware: "Firmware", serial: "Serial Number",
    interface: "Interface", transfer_mode: "Transfer Mode", drive_map: "Drive",
    standard: "Standard", feature: "Feature", buffer_size: "Buffer Size", nv_cache: "NV Cache",
    power_on: "Power On Hours", power_on_count: "Power On Count",
    rotation: "Rotation Rate", rpm: "RPM", ssd: "SSD", device: "Device", smart: "SMART",
    life: "Life", status_reasons: "Status",
    rescan: "Rescan", updated: "Updated", theme: "Theme",
    theme_classic: "Light", theme_dark: "Dark", theme_follow: "Follow fnOS", lang: "Language",
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
  },
  "zh-CN": {
    good: "良好", caution: "注意", bad: "损坏", unknown: "未知", ok: "正常",
    health: "健康状态", temperature: "温度", firmware: "固件", serial: "序列号",
    interface: "接口", transfer_mode: "传输模式", drive_map: "设备",
    standard: "标准", feature: "特性", buffer_size: "缓存大小", nv_cache: "NV 缓存",
    power_on: "通电时间", power_on_count: "通电次数",
    rotation: "转速", rpm: "RPM", ssd: "固态", device: "设备", smart: "SMART",
    life: "寿命", status_reasons: "状态",
    rescan: "重新扫描", updated: "更新于", theme: "主题",
    theme_classic: "浅色", theme_dark: "深色", theme_follow: "跟随 fnOS", lang: "语言",
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
  },
  "zh-TW": {
    good: "良好", caution: "注意", bad: "損壞", unknown: "未知", ok: "正常",
    health: "健康狀態", temperature: "溫度", firmware: "韌體", serial: "序號",
    interface: "介面", transfer_mode: "傳輸模式", drive_map: "裝置",
    standard: "標準", feature: "特性", buffer_size: "緩衝區大小", nv_cache: "NV 快取",
    power_on: "通電時間", power_on_count: "通電次數",
    rotation: "轉速", rpm: "RPM", ssd: "固態", device: "裝置", smart: "SMART",
    life: "壽命", status_reasons: "狀態",
    rescan: "重新掃描", updated: "更新於", theme: "主題",
    theme_classic: "淺色", theme_dark: "深色", theme_follow: "跟隨 fnOS", lang: "語言",
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
  },
  ja: {
    good: "良好", caution: "注意", bad: "異常", unknown: "不明", ok: "正常",
    health: "健康状態", temperature: "温度", firmware: "ファームウェア", serial: "シリアル番号",
    interface: "インターフェース", transfer_mode: "転送モード", drive_map: "ドライブ",
    standard: "標準", feature: "機能", buffer_size: "バッファサイズ", nv_cache: "NV キャッシュ",
    power_on: "通電時間", power_on_count: "通電回数",
    rotation: "回転数", rpm: "RPM", ssd: "SSD", device: "デバイス", smart: "S.M.A.R.T.",
    life: "寿命", status_reasons: "状態",
    rescan: "再スキャン", updated: "更新", theme: "テーマ",
    theme_classic: "ライト", theme_dark: "ダーク", theme_follow: "fnOS に従う", lang: "言語",
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
  }
};

// Menu / dialog strings added with the CDI-style menubar.
const EXTRA = {
  en: {
    menu_file: "File", menu_edit: "Edit", menu_function: "Function", menu_theme: "Theme",
    menu_disk: "Disk", menu_help: "Help",
    save_text: "Save Text", copy: "Copy", refresh: "Refresh", auto_refresh: "Auto Refresh",
    disable: "Disable", hide_serial_number: "Hide Serial Number", alerts: "Alerts",
    alarm_list: "Alarm List", advanced: "Advanced",
    health_status_setting: "Health Status Setting", temperature_setting: "Temperature Setting",
    temp_unit: "Temperature Unit", raw_values: "Raw Values", hide_smart: "Hide S.M.A.R.T. Information",
    hide_no_smart: "Hide No S.M.A.R.T. Disk", disk_sort: "Drive Sort Method",
    sort_device: "Device", sort_model: "Model", sort_serial: "Serial",
    zoom: "Zoom", zoom_auto: "Auto", font_setting: "Font Setting", font_family: "Font",
    font_size: "Size", font_default: "Default",
    about: "About", about_smart: "About S.M.A.R.T.",
    about_text: "DiskInfo for fnOS — a CrystalDiskInfo-style disk health viewer.\nInspired by CrystalDiskInfo (MIT). SMART data via smartmontools.",
    theme_import: "Import Theme Pack...", theme_delete: "Delete Theme...",
    theme_import_ok: "Theme imported", theme_import_fail: "Import failed",
    theme_delete_confirm: "Delete this theme?", theme_custom: "Custom Theme",
    copy_ok: "Copied to clipboard", copy_fail: "Copy failed",
    aam_apm_unsupported: "Device does not report AAM/APM support",
    g_options: "Graph Options", g_legend: "Legend", g_timeformat: "Time Format",
    g_weekend: "Paint Weekend", g_background: "Background Image",
    g_use_theme: "Use theme background", g_color: "Line Colors",
    about_graph: "Graph/Option UI adapted from CrystalDiskInfo (MIT).",
    fallback_theme: "Fallback Theme", none: "None", auto: "Auto",
    total_host_reads: "Total Host Reads", total_host_writes: "Total Host Writes",
    total_nand_writes: "Total NAND Writes", years: "years", days: "days",
    tip_transfer_mode: "Current Mode | Supported Mode",
    tip_standard: "Major Version | Minor Version",
    tip_feature: "S.M.A.R.T.: Self-Monitoring, Analysis and Reporting Technology\nAPM: Advanced Power Management\nAAM: Automatic Acoustic Management\nNCQ: Native Command Queuing\nTRIM: Trim function of DATA SET MANAGEMENT command\nDevSleep: Device Sleep\nStreaming: Streaming Feature Set\nGPL: General Purpose Log",
    query: "Read", save_image: "Save Image", auto_refresh_target: "Auto Refresh Target",
    not_available: "Device does not provide AAM/APM data", disabled: "disabled", raw_output: "Raw output",
    value_required: "Enter a value between 1 and 254"
  },
  "zh-CN": {
    menu_file: "文件", menu_edit: "编辑", menu_function: "功能", menu_theme: "主题",
    menu_disk: "硬盘", menu_help: "帮助",
    save_text: "保存文本", copy: "复制", refresh: "刷新", auto_refresh: "自动刷新",
    disable: "禁用", hide_serial_number: "隐藏序列号", alerts: "报警功能",
    alarm_list: "报警列表", advanced: "高级功能",
    health_status_setting: "健康状态设置", temperature_setting: "温度设置",
    temp_unit: "温度单位", raw_values: "原始值显示", hide_smart: "隐藏 S.M.A.R.T. 信息",
    hide_no_smart: "隐藏无 S.M.A.R.T. 硬盘", disk_sort: "硬盘排序",
    sort_device: "设备名", sort_model: "型号", sort_serial: "序列号",
    zoom: "缩放", zoom_auto: "自动", font_setting: "字体设置", font_family: "字体",
    font_size: "字号", font_default: "默认",
    about: "关于", about_smart: "关于 S.M.A.R.T.",
    about_text: "DiskInfo for fnOS —— CrystalDiskInfo 风格的硬盘健康查看器。\n灵感来自 CrystalDiskInfo（MIT）。SMART 数据来自 smartmontools。",
    theme_import: "导入主题包…", theme_delete: "删除主题…",
    theme_import_ok: "主题已导入", theme_import_fail: "导入失败",
    theme_delete_confirm: "确定删除该主题？", theme_custom: "自定义主题",
    copy_ok: "已复制到剪贴板", copy_fail: "复制失败",
    aam_apm_unsupported: "设备未上报 AAM/APM 支持",
    g_options: "图表选项", g_legend: "图例", g_timeformat: "时间格式",
    g_weekend: "标注周末", g_background: "背景图片",
    g_use_theme: "使用主题背景", g_color: "折线颜色",
    about_graph: "图表/选项界面移植自 CrystalDiskInfo（MIT）。",
    fallback_theme: "回退主题", none: "无", auto: "自动",
    total_host_reads: "主机总计读取", total_host_writes: "主机总计写入",
    total_nand_writes: "NAND 总计写入", years: "年", days: "天",
    tip_transfer_mode: "当前的传输模式 | 支持的传输模式",
    tip_standard: "主要版本 | 次要版本",
    tip_feature: "S.M.A.R.T.：自我监测、分析与报告技术\nAPM：高级电源管理\nAAM：自动噪声管理\nNCQ：原生命令队列\nTRIM：DATA SET MANAGEMENT 命令的 Trim 功能\nDevSleep：设备睡眠\nStreaming：流式传输特性集\nGPL：通用用途日志",
    query: "读取", save_image: "保存图片", auto_refresh_target: "自动刷新对象",
    not_available: "设备未提供 AAM/APM 数据", disabled: "已禁用", raw_output: "原始输出",
    value_required: "请输入 1-254 之间的数值"
  },
  "zh-TW": {
    menu_file: "檔案", menu_edit: "編輯", menu_function: "功能", menu_theme: "主題",
    menu_disk: "硬碟", menu_help: "說明",
    save_text: "儲存文字", copy: "複製", refresh: "重新整理", auto_refresh: "自動重新整理",
    disable: "停用", hide_serial_number: "隱藏序號", alerts: "警報功能",
    alarm_list: "警報列表", advanced: "進階功能",
    health_status_setting: "健康狀態設定", temperature_setting: "溫度設定",
    temp_unit: "溫度單位", raw_values: "原始值顯示", hide_smart: "隱藏 S.M.A.R.T. 資訊",
    hide_no_smart: "隱藏無 S.M.A.R.T. 硬碟", disk_sort: "硬碟排序",
    sort_device: "裝置名稱", sort_model: "型號", sort_serial: "序號",
    zoom: "縮放", zoom_auto: "自動", font_setting: "字型設定", font_family: "字型",
    font_size: "字級", font_default: "預設",
    about: "關於", about_smart: "關於 S.M.A.R.T.",
    about_text: "DiskInfo for fnOS —— CrystalDiskInfo 風格的硬碟健康檢視器。\n靈感來自 CrystalDiskInfo（MIT）。SMART 資料來自 smartmontools。",
    theme_import: "匯入主題包…", theme_delete: "刪除主題…",
    theme_import_ok: "主題已匯入", theme_import_fail: "匯入失敗",
    theme_delete_confirm: "確定刪除該主題？", theme_custom: "自訂主題",
    copy_ok: "已複製到剪貼簿", copy_fail: "複製失敗",
    aam_apm_unsupported: "裝置未回報 AAM/APM 支援",
    g_options: "圖表選項", g_legend: "圖例", g_timeformat: "時間格式",
    g_weekend: "標註週末", g_background: "背景圖片",
    g_use_theme: "使用主題背景", g_color: "折線顏色",
    about_graph: "圖表/選項介面移植自 CrystalDiskInfo（MIT）。",
    fallback_theme: "回退主題", none: "無", auto: "自動",
    total_host_reads: "對 SSD 累計讀取", total_host_writes: "對 SSD 累計寫入",
    total_nand_writes: "NAND 累計寫入", years: "年", days: "天",
    tip_transfer_mode: "目前模式 | 支援的模式",
    tip_standard: "主要版本 | 次要版本",
    tip_feature: "S.M.A.R.T.：自我監視、分析與報告技術\nAPM：進階電源管理\nAAM：自動噪音管理\nNCQ：原生指令佇列\nTRIM：DATA SET MANAGEMENT 指令的 Trim 功能\nDevSleep：裝置睡眠\nStreaming：串流功能集\nGPL：一般用途記錄",
    query: "讀取", save_image: "儲存圖片", auto_refresh_target: "自動重新整理對象",
    not_available: "裝置未提供 AAM/APM 資料", disabled: "已停用", raw_output: "原始輸出",
    value_required: "請輸入 1-254 之間的數值"
  },
  ja: {
    menu_file: "ファイル", menu_edit: "編集", menu_function: "機能", menu_theme: "テーマ",
    menu_disk: "ディスク", menu_help: "ヘルプ",
    save_text: "テキストを保存", copy: "コピー", refresh: "更新", auto_refresh: "自動更新",
    disable: "無効", hide_serial_number: "シリアル番号を隠す", alerts: "アラーム機能",
    alarm_list: "アラーム一覧", advanced: "詳細機能",
    health_status_setting: "健康状態設定", temperature_setting: "温度設定",
    temp_unit: "温度単位", raw_values: "生値表示", hide_smart: "S.M.A.R.T. 情報を隠す",
    hide_no_smart: "S.M.A.R.T. 非対応ディスクを隠す", disk_sort: "ディスクの並べ替え",
    sort_device: "デバイス名", sort_model: "型番", sort_serial: "シリアル番号",
    zoom: "ズーム", zoom_auto: "自動", font_setting: "フォント設定", font_family: "フォント",
    font_size: "文字サイズ", font_default: "デフォルト",
    about: "バージョン情報", about_smart: "S.M.A.R.T. について",
    about_text: "DiskInfo for fnOS — CrystalDiskInfo スタイルのディスク健康ビューア。\nCrystalDiskInfo (MIT) にインスパイアされました。SMART データは smartmontools によるものです。",
    theme_import: "テーマをインポート…", theme_delete: "テーマを削除…",
    theme_import_ok: "テーマをインポートしました", theme_import_fail: "インポートに失敗しました",
    theme_delete_confirm: "このテーマを削除しますか？", theme_custom: "カスタムテーマ",
    copy_ok: "クリップボードにコピーしました", copy_fail: "コピーに失敗しました",
    aam_apm_unsupported: "AAM/APM 非対応のデバイスです",
    g_options: "グラフ設定", g_legend: "凡例", g_timeformat: "時刻形式",
    g_weekend: "週末を塗る", g_background: "背景画像",
    g_use_theme: "テーマ背景を使用", g_color: "線の色",
    about_graph: "グラフ/オプション UI は CrystalDiskInfo (MIT) から移植しました。",
    fallback_theme: "フォールバックテーマ", none: "なし", auto: "自動",
    total_host_reads: "総読込量 (ホスト)", total_host_writes: "総書込量 (ホスト)",
    total_nand_writes: "総書込量 (NAND)", years: "年", days: "日",
    tip_transfer_mode: "現在の転送モード | 対応転送モード",
    tip_standard: "メジャーバージョン | マイナーバージョン",
    tip_feature: "S.M.A.R.T.: 自己監視・分析・報告技術\nAPM: 高度電源管理\nAAM: 自動音響管理\nNCQ: ネイティブコマンドキューイング\nTRIM: DATA SET MANAGEMENT コマンドの Trim 機能\nDevSleep: デバイススリープ\nStreaming: ストリーミング機能セット\nGPL: 汎用ログ",
    query: "読み取り", save_image: "画像を保存", auto_refresh_target: "自動更新の対象",
    not_available: "デバイスは AAM/APM 情報を提供していません", disabled: "無効", raw_output: "生の出力",
    value_required: "1〜254 の値を入力してください"
  }
};

for (const [lang, map] of Object.entries(EXTRA)) {
  Object.assign(I18N[lang], map);
}

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

// attrName resolves a SMART attribute name via CrystalDiskInfo's tables
// (attr-i18n.js, all [Smart*] sections). The disk's matched section wins,
// then the kind default, then Smart; unknown IDs fall back to smartctl's
// English name (Aa_Bb -> Aa Bb).
function attrName(id, fallback, smartKey, isSsd, isNvme) {
  const hex = id.toString(16).toUpperCase().padStart(2, "0");
  const tables = typeof ATTR_I18N !== "undefined" ? ATTR_I18N[LANG] || {} : {};
  const chain = [smartKey];
  if (isNvme) {
    chain.push("SmartNVMe");
  }
  if (isSsd) {
    chain.push("SmartSsd");
  }
  chain.push("Smart");
  for (const key of chain) {
    const tbl = key ? tables[key] : null;
    if (tbl && tbl[hex]) {
      return tbl[hex];
    }
  }
  return (fallback || "ID " + hex).replace(/_/g, " ");
}
