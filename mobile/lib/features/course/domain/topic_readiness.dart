class TopicReadiness {
  final bool isReady;
  final int validatedCount;
  final int totalCount;
  final String message;

  const TopicReadiness({
    required this.isReady,
    required this.validatedCount,
    required this.totalCount,
    required this.message,
  });

  factory TopicReadiness.fromJson(Map<String, dynamic> json) {
    return TopicReadiness(
      isReady: (json['is_ready'] as bool?) ?? false,
      validatedCount: (json['validated_count'] as num?)?.toInt() ?? 0,
      totalCount: (json['total_count'] as num?)?.toInt() ?? 0,
      message: (json['message'] as String?) ?? '',
    );
  }
}
