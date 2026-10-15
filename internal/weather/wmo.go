package weather

type wmoText struct{ zh, en string }

var wmoCodes = map[int]wmoText{
	0: {"晴", "Clear sky"}, 1: {"大部晴朗", "Mainly clear"}, 2: {"多云", "Partly cloudy"}, 3: {"阴", "Overcast"},
	45: {"雾", "Fog"}, 48: {"雾凇", "Depositing rime fog"},
	51: {"小毛毛雨", "Light drizzle"}, 53: {"毛毛雨", "Moderate drizzle"}, 55: {"大毛毛雨", "Dense drizzle"},
	56: {"冻毛毛雨", "Light freezing drizzle"}, 57: {"强冻毛毛雨", "Dense freezing drizzle"},
	61: {"小雨", "Slight rain"}, 63: {"中雨", "Moderate rain"}, 65: {"大雨", "Heavy rain"},
	66: {"冻雨", "Light freezing rain"}, 67: {"强冻雨", "Heavy freezing rain"},
	71: {"小雪", "Slight snow"}, 73: {"中雪", "Moderate snow"}, 75: {"大雪", "Heavy snow"}, 77: {"米雪", "Snow grains"},
	80: {"小阵雨", "Slight rain showers"}, 81: {"阵雨", "Moderate rain showers"}, 82: {"强阵雨", "Violent rain showers"},
	85: {"小阵雪", "Slight snow showers"}, 86: {"大阵雪", "Heavy snow showers"},
	95: {"雷阵雨", "Thunderstorm"}, 96: {"雷阵雨伴小冰雹", "Thunderstorm with slight hail"}, 99: {"雷阵雨伴大冰雹", "Thunderstorm with heavy hail"},
}

// DescribeWMO returns Chinese and English text for a WMO weather code.
func DescribeWMO(code int) (string, string) {
	if t, ok := wmoCodes[code]; ok {
		return t.zh, t.en
	}
	return "未知", "Unknown"
}
