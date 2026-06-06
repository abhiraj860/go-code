import math
from langchain_core.tools import tool


@tool
def calculate(expression: str) -> str:
    """Evaluate a mathematical expression and return the numeric result.

    Supports standard arithmetic (+, -, *, /), powers (**), and common math
    functions: sqrt, floor, ceil, round, abs, min, max.

    Args:
        expression: A mathematical expression string, e.g. "5570 / 900" or "round(6.19 * 60, 1)"

    Returns:
        The numeric result as a string, or an error message.

    Note:
        Uses eval() with a restricted namespace. Suitable for demo purposes only —
        do not pass untrusted user input directly to this function in production.
    """
    safe_builtins = {
        name: getattr(math, name)
        for name in dir(math)
        if not name.startswith("__")
    }
    safe_builtins.update({
        "abs": abs, "round": round, "int": int, "float": float,
        "min": min, "max": max,
    })
    try:
        result = eval(expression, {"__builtins__": {}}, safe_builtins)
        if isinstance(result, float):
            return str(round(result, 6))
        return str(result)
    except Exception as e:
        return f"Error evaluating '{expression}': {e}"


# Verify the tool works before connecting it to the agent
print("calculate('5570 / 900') →", calculate.invoke({"expression": "5570 / 900"}))
print("calculate('round(6.188889 * 60, 2)') →", calculate.invoke({"expression": "round(6.188889 * 60, 2)"}))





@tool
def convert_units(value: float, from_unit: str, to_unit: str) -> str:
    """Convert a numeric value from one unit to another.

    Supported unit groups:
      Distance : km, miles, meters, feet, nautical miles
      Speed    : km/h, mph, m/s, knots
      Time     : hours, minutes, seconds

    Units within the same group can be converted freely.
    Cross-group conversions (e.g. km to hours) will return an error.

    Args:
        value     : The numeric value to convert
        from_unit : Source unit, e.g. "km/h", "miles", "hours"
        to_unit   : Target unit, e.g. "mph", "km",   "minutes"

    Returns:
        The converted value with target unit label, or an error message.
    """
    # Conversion factors relative to base unit per group
    # (distance base = km, speed base = km/h, time base = hours)
    to_base = {
        "km": 1.0,          "miles": 1.60934,    "meters": 0.001,
        "feet": 0.0003048,  "nautical miles": 1.852,
        "km/h": 1.0,        "mph": 1.60934,      "m/s": 3.6,
        "knots": 1.852,
        "hours": 1.0,       "minutes": 1 / 60,   "seconds": 1 / 3600,
    }
    groups = {
        "distance": {"km", "miles", "meters", "feet", "nautical miles"},
        "speed":    {"km/h", "mph", "m/s", "knots"},
        "time":     {"hours", "minutes", "seconds"},
    }

    f, t = from_unit.lower().strip(), to_unit.lower().strip()

    from_grp = next((g for g, units in groups.items() if f in units), None)
    to_grp   = next((g for g, units in groups.items() if t in units), None)

    if from_grp is None:
        return f"Unknown unit '{from_unit}'. Supported: {', '.join(sorted(to_base))}"
    if to_grp is None:
        return f"Unknown unit '{to_unit}'. Supported: {', '.join(sorted(to_base))}"
    if from_grp != to_grp:
        return f"Cannot convert {from_grp} unit '{from_unit}' to {to_grp} unit '{to_unit}'."

    result = value * to_base[f] / to_base[t]
    return f"{round(result, 4)} {to_unit}"


print("convert_units(6.188889, 'hours', 'minutes') →",
      convert_units.invoke({"value": 6.188889, "from_unit": "hours", "to_unit": "minutes"}))
print("convert_units(900, 'km/h', 'mph') →",
      convert_units.invoke({"value": 900, "from_unit": "km/h", "to_unit": "mph"}))





@tool
def timezone_convert(time_str: str, from_city: str, to_city: str) -> str:
    """Convert a local time from one city's timezone to another.

    Uses standard (non-DST) UTC offsets. Supported cities include:
      UTC+0  : London, Dublin, Accra
      UTC+1  : Paris, Berlin, Amsterdam, Rome, Madrid
      UTC+2  : Cairo, Helsinki, Athens
      UTC+3  : Moscow, Riyadh, Nairobi
      UTC+4  : Dubai
      UTC+5.5: Mumbai, New Delhi, Kolkata
      UTC+8  : Singapore, Hong Kong, Beijing, Shanghai, Perth
      UTC+9  : Tokyo, Seoul
      UTC+10 : Sydney, Melbourne, Brisbane
      UTC-3  : São Paulo, Buenos Aires
      UTC-5  : New York, Toronto, Lima
      UTC-6  : Chicago, Mexico City
      UTC-8  : Los Angeles, Vancouver, Seattle
      UTC-10 : Honolulu

    Args:
        time_str  : Local time in HH:MM (24-hour format), e.g. "14:00"
        from_city : Source city name (case-insensitive)
        to_city   : Destination city name (case-insensitive)

    Returns:
        The equivalent local time in the destination city, with a day offset note
        if the conversion crosses midnight.
    """
    offsets = {
        "london": 0, "dublin": 0, "accra": 0,
        "paris": 1, "berlin": 1, "amsterdam": 1, "rome": 1, "madrid": 1,
        "cairo": 2, "helsinki": 2, "athens": 2,
        "moscow": 3, "riyadh": 3, "nairobi": 3,
        "dubai": 4,
        "mumbai": 5.5, "new delhi": 5.5, "kolkata": 5.5,
        "singapore": 8, "hong kong": 8, "beijing": 8, "shanghai": 8, "perth": 8,
        "tokyo": 9, "seoul": 9,
        "sydney": 10, "melbourne": 10, "brisbane": 10,
        "auckland": 12,
        "sao paulo": -3, "são paulo": -3, "buenos aires": -3,
        "new york": -5, "toronto": -5, "lima": -5,
        "chicago": -6, "mexico city": -6,
        "los angeles": -8, "vancouver": -8, "seattle": -8,
        "honolulu": -10,
    }

    f = from_city.lower().strip()
    t = to_city.lower().strip()

    if f not in offsets:
        return f"Unknown city '{from_city}'. See the tool docstring for supported cities."
    if t not in offsets:
        return f"Unknown city '{to_city}'. See the tool docstring for supported cities."

    try:
        h, m = map(int, time_str.strip().split(":"))
    except ValueError:
        return f"Invalid time format '{time_str}'. Use HH:MM (24-hour), e.g. '14:00'."

    total_min  = h * 60 + m
    utc_min    = total_min - int(offsets[f] * 60)
    dest_min   = utc_min   + int(offsets[t] * 60)

    day_note = ""
    if dest_min >= 1440:
        dest_min -= 1440
        day_note = " (next day)"
    elif dest_min < 0:
        dest_min += 1440
        day_note = " (previous day)"

    dh, dm = dest_min // 60, dest_min % 60
    to_off  = offsets[t]

    return (
        f"{dh:02d}:{dm:02d} local time in {to_city.title()}"
        f" (UTC{to_off:+.4g}){day_note}"
    )


print("timezone_convert('20:11', 'London', 'New York') →",
      timezone_convert.invoke({"time_str": "20:11", "from_city": "London", "to_city": "New York"}))
print("timezone_convert('14:00', 'London', 'Tokyo') →",
      timezone_convert.invoke({"time_str": "14:00", "from_city": "London", "to_city": "Tokyo"}))