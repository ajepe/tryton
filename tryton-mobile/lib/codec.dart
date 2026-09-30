const String classKey = '__class__';

dynamic encodeValue(dynamic value) {
  if (value == null) return null;
  if (value is DateTime) {
    return {
      classKey: 'datetime',
      'year': value.year,
      'month': value.month,
      'day': value.day,
      'hour': value.hour,
      'minute': value.minute,
      'second': value.second,
    };
  }
  if (value is List) {
    return value.map(encodeValue).toList();
  }
  if (value is Map) {
    return value.map((key, item) => MapEntry(key.toString(), encodeValue(item)));
  }
  return value;
}

dynamic decodeValue(dynamic value) {
  if (value == null) return null;
  if (value is List) {
    return value.map(decodeValue).toList();
  }
  if (value is Map) {
    final className = value[classKey];
    switch (className) {
      case 'datetime':
        return DateTime(
          value['year'] as int,
          value['month'] as int,
          value['day'] as int,
          value['hour'] as int,
          value['minute'] as int,
          value['second'] as int,
        );
      case 'date':
        return DateTime(value['year'] as int, value['month'] as int, value['day'] as int);
      case 'timedelta':
        return Duration(seconds: (value['seconds'] as num).toInt());
      case 'Decimal':
        return num.tryParse(value['decimal'].toString()) ?? 0;
      default:
        return value.map((key, item) => MapEntry(key.toString(), decodeValue(item)));
    }
  }
  return value;
}
