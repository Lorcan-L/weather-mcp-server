# 发布说明 v1.4.0（中文版首发）

基于 [ezh0v/weather-mcp-server](https://github.com/ezh0v/weather-mcp-server)（MIT，作者 Ezhov Artem），针对中文用户做了大幅改造。

## 新功能
- 默认改用 Open-Meteo，**不再需要 API Key**
- 支持中文城市名（如"厦门"），会自动尝试加"市/区/县"后缀的写法，并优先返回中国境内的地点
- 实时天气 + 1–16 天逐日预报，中英双语输出（`lang=zh|en|both`）
- 空气质量：按 HJ 633-2012 估算中国 AQI（含等级和首要污染物），同时给出 US/EU AQI 和各污染物浓度
- 二十四节气：查询某天的当前节气和下一个节气，或列出某年全部节气（北京时间，精确到分钟）
- 可选接入和风天气 QWeather（`QWEATHER_API_KEY` + `QWEATHER_API_HOST`）
- 新增 Streamable HTTP 传输（`--http :8000` → `/mcp`），保留 stdio 和 SSE
- 新 Logo（`assets/logo.png` / `assets/logo.svg`）

## 变更
- 升级 mcp-go 到 v1.3.0，Go 版本要求 ≥ 1.25
- 移除原来的 WeatherAPI.com 数据源和 mock 代码；原作者版权保留在 LICENSE 中

## 已知限制
- 中国 AQI 是模型估算值，不是国控站点实测数据
- 不配置 QWeather 时，区县级地名识别率较低
- 安装请用 `@master`：仓库沿用了上游的 v1.3.0 标签，它的模块路径是原作者的
