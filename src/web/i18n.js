"use strict";

const I18N = {
  en: {
    good: "GOOD", caution: "CAUTION", bad: "BAD", unknown: "UNKNOWN",
    ok: "OK",
    health: "Health", temperature: "Temperature",
    firmware: "Firmware", serial: "Serial Number", interface: "Interface",
    power_on: "Power On Hours", rotation: "Rotation Rate", rpm: "RPM",
    ssd: "SSD", device: "Device", smart: "SMART",
    rescan: "Rescan", updated: "Updated", theme: "Theme",
    theme_classic: "Classic", theme_dark: "Dark", theme_follow: "Follow fnOS",
    col_id: "ID", col_attr: "Attribute", col_cur: "Current", col_worst: "Worst",
    col_thr: "Threshold", col_raw: "Raw", col_status: "Status",
    no_disks: "No disks found", no_attrs: "No SMART attributes", hours: "hours"
  },
  "zh-CN": {
    good: "良好", caution: "注意", bad: "损坏", unknown: "未知",
    ok: "正常",
    health: "健康状态", temperature: "温度",
    firmware: "固件", serial: "序列号", interface: "接口",
    power_on: "通电时间", rotation: "转速", rpm: "RPM",
    ssd: "固态", device: "设备", smart: "SMART",
    rescan: "重新扫描", updated: "更新于", theme: "主题",
    theme_classic: "经典", theme_dark: "深色", theme_follow: "跟随 fnOS",
    col_id: "ID", col_attr: "属性", col_cur: "当前", col_worst: "最差",
    col_thr: "阈值", col_raw: "原始值", col_status: "状态",
    no_disks: "未发现硬盘", no_attrs: "无 SMART 属性", hours: "小时",
    attr_1: "底层读取错误率", attr_3: "旋转时间", attr_4: "启动次数", attr_5: "重映射扇区计数",
    attr_7: "寻道错误率", attr_9: "通电时间", attr_10: "旋转重试计数", attr_12: "断电启动次数",
    attr_13: "读取软错误率", attr_183: "运行中降级事件", attr_184: "端到端错误",
    attr_187: "报告的不可纠正错误", attr_188: "命令超时", attr_190: "气流温度",
    attr_192: "断电退刀次数", attr_193: "磁头加载次数", attr_194: "温度",
    attr_196: "重映射事件计数", attr_197: "当前待映射扇区", attr_198: "离线不可纠正扇区",
    attr_199: "UDMA CRC 错误", attr_200: "多区域错误", attr_240: "磁头飞行时间",
    attr_241: "写入 LBA 总数", attr_242: "读取 LBA 总数"
  }
};

const LANG = (navigator.language || "en").toLowerCase().startsWith("zh") ? "zh-CN" : "en";

function t(key) {
  const m = I18N[LANG] || {};
  return m[key] || I18N.en[key] || key;
}

function attrName(id, fallback) {
  const m = I18N[LANG] || {};
  return m["attr_" + id] || (fallback || "").replace(/_/g, " ");
}
