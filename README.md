<div align="center">
<img alt="weather" src="assets/weather.svg">

<h1>Weather MCP Server · 中文版 (CN)</h1>

[![License](https://img.shields.io/badge/license-MIT-red.svg)](LICENSE)
[![Go Version](https://img.shields.io/github/go-mod/go-version/Lorcan-L/weather-mcp-server)](go.mod)

**中文** | [English](#english)

</div>

> **致谢 / Credits**：本项目 fork 自 [ezh0v/weather-mcp-server](https://github.com/ezh0v/weather-mcp-server)（作者 **Ezhov Artem**，MIT 许可）。原作者的版权声明完整保留在 [LICENSE](LICENSE) 中。本 fork 重写了数据层，加入面向中文用户的功能，但沿用了原项目的思路、工程结构（Go + mcp-go、stdio/SSE 双模式）和 CI 配置。感谢原作者！
> This is a fork of ezh0v/weather-mcp-server by Ezhov Artem (MIT). The original copyright notice is kept in LICENSE.

## 简介

这是一个轻量的 MCP（Model Context Protocol）天气服务器，主要面向中文用户：

- 🆓 **默认无需 API Key**：使用 [Open-Meteo](https://open-meteo.com/)（免费、免注册）
- 🀄 **中文城市名**：直接输入"厦门"、"北京"、"杭州市"即可，也支持英文 "Xiamen"。会自动尝试"厦门 / 厦门市"等写法，并优先返回中国境内的结果
- 🌤 **实时天气 + 1–16 天预报**，天气现象输出中文
- 😷 **空气质量**：按 HJ 633-2012 标准从污染物浓度**估算中国 AQI**，给出等级和首要污染物，同时提供 US/EU AQI 和 PM2.5、PM10、O₃ 等指标
- 🌾 **二十四节气**：查询某天所处的节气和下一个节气（精确到分钟，北京时间），或列出某年全部节气。本地天文算法计算，不需要联网
- 🌐 **中英双语输出**：`lang=zh | en | both`（默认 both）
- 🔑 **可选和风天气 QWeather**：设置环境变量后，天气和预报改由和风天气提供，区县级地名识别更好
- 🔌 **传输方式**：stdio（本地）、Streamable HTTP（`/mcp`），并保留原有的 SSE

### 工具列表

| 工具 | 说明 | 参数 |
|---|---|---|
| `search_city` | 按中文或英文名称搜索地点 | `city` |
| `get_current_weather` | 实时天气 | `city`, `lang` |
| `get_forecast` | 逐日预报 | `city`, `days` (1–16), `lang` |
| `get_air_quality` | 空气质量（中国 AQI 估算值 / US / EU） | `city`, `lang` |
| `get_solar_term` | 二十四节气 | `date` (YYYY-MM-DD) 或 `year`, `lang` |

示例输出（2026-10-15，厦门）：

```
厦门市, 福建省, 中国 — 实时天气 / Current weather (2026-10-15T14:15)
天气 / Condition: 多云 / Partly cloudy
气温 / Temperature: 29.8°C (体感 / feels like 30.3°C)
...
2026-10-15 所处节气 / current term: 寒露 / Cold Dew (始于 / since 2026-10-08)
下一个节气 / Next term: 霜降 / Frost's Descent — 2026-10-23 17:39 CST (8 天后 / days)
```

## 安装

需要 Go 1.25 及以上版本：

```bash
go install github.com/Lorcan-L/weather-mcp-server/cmd/weather-mcp-server@latest
# 二进制安装在 $(go env GOPATH)/bin/weather-mcp-server
```

也可以用 Docker（Streamable HTTP，端口 8000）：

```bash
docker build -t weather-mcp-cn . && docker run -p 8000:8000 weather-mcp-cn
# MCP 地址: http://localhost:8000/mcp
```

## 在 Cursor / Grok Bot 中配置

**Cursor**：编辑 `~/.cursor/mcp.json`（全局）或项目内的 `.cursor/mcp.json`：

```json
{
  "mcpServers": {
    "weather-cn": {
      "command": "/absolute/path/to/weather-mcp-server"
    }
  }
}
```

**远程 / HTTP 模式**（适用于 Grok Bot、Cursor 及其他支持 Streamable HTTP 的客户端）：先运行 `weather-mcp-server --http :8000`，再添加：

```json
{
  "mcpServers": {
    "weather-cn": { "url": "http://localhost:8000/mcp" }
  }
}
```

**Grok Bot**：可以在插件市场中安装；也可以在 MCP 设置中按上面的格式填写本地命令或 HTTP 地址。

### 可选：使用和风天气 QWeather

```json
{
  "mcpServers": {
    "weather-cn": {
      "command": "/absolute/path/to/weather-mcp-server",
      "env": {
        "QWEATHER_API_KEY": "你的 key",
        "QWEATHER_API_HOST": "xxxx.re.qweatherapi.com"
      }
    }
  }
}
```

`QWEATHER_API_HOST` 是和风控制台分配给你的专属 API Host。空气质量和节气功能始终不需要 key。

## 说明与局限

- 中国 AQI 是根据 Open-Meteo（CAMS 模型）的污染物浓度估算的，**不是**国控站点的实测数据，仅供参考
- 不配置 QWeather 时，地名精确到城市级效果最好；区县、街道级地名建议配置 QWeather
- 节气时间根据太阳视黄经（Meeus 算法）计算，与官方公布的时间误差通常在几分钟内

## 开发

```bash
make run-tests   # go test -v -race ./...
go run ./cmd/weather-mcp-server            # stdio
go run ./cmd/weather-mcp-server --http :8000  # Streamable HTTP, /mcp
```

---

<a id="english"></a>
## English

A lightweight MCP weather server for Chinese users. It is a fork of **[ezh0v/weather-mcp-server](https://github.com/ezh0v/weather-mcp-server)** by Ezhov Artem (MIT); the original copyright is kept in [LICENSE](LICENSE).

**Features**
- **No API key by default**: uses Open-Meteo for geocoding, weather and air quality
- **Chinese place names**: 厦门, 北京 and 杭州市 work, as do English names. The server tries name variants (e.g. adding or removing 市) and prefers results in China
- Current weather and a 1–16 day forecast, with conditions in Chinese and English
- **Air quality**: an estimated China AQI (HJ 633-2012) with its category and primary pollutant, plus US/EU AQI and pollutant concentrations
- **24 solar terms (二十四节气)**: the current and next term for any date, or every term in a year. Times are in China Standard Time and computed locally from solar longitude
- Bilingual output via `lang=zh|en|both`
- Optional QWeather provider: set `QWEATHER_API_KEY` and `QWEATHER_API_HOST`
- Transports: stdio (default), Streamable HTTP (`--http :8000` → `/mcp`), and legacy SSE (`--address`)

**Install**: `go install github.com/Lorcan-L/weather-mcp-server/cmd/weather-mcp-server@latest`. The configuration for Cursor and Grok Bot is the same as in the Chinese section above.

**Limitations**: the China AQI is a model-based estimate, not data from official monitoring stations. District-level names resolve better when QWeather is configured.

## License

MIT. Copyright (c) 2025 Ezhov Artem (original work). Modifications are released under the same MIT license.
