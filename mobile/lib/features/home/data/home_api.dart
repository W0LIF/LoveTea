import '../../../core/network/api_client.dart';

class HomeSnapshot {
  final String userId;
  final String email;
  final String? displayName;
  final String? coupleStatus;
  final List<HomeMember> members;
  final HomePet? pet;

  const HomeSnapshot({
    required this.userId,
    required this.email,
    required this.displayName,
    required this.coupleStatus,
    required this.members,
    required this.pet,
  });

  factory HomeSnapshot.fromJson(Map<String, dynamic> json) {
    final profile = Map<String, dynamic>.from(json['profile'] as Map);
    final coupleData = json['couple'];
    final couple = coupleData is Map
        ? Map<String, dynamic>.from(coupleData)
        : null;
    final rawMembers = couple?['members'];
    final members = rawMembers is List
        ? rawMembers
              .whereType<Map>()
              .map(
                (member) =>
                    HomeMember.fromJson(Map<String, dynamic>.from(member)),
              )
              .toList()
        : <HomeMember>[];
    final petData = json['pet'];
    final pet = petData is Map
        ? HomePet.fromJson(Map<String, dynamic>.from(petData))
        : null;

    return HomeSnapshot(
      userId: profile['user_id'] as String,
      email: profile['email'] as String? ?? '',
      displayName: profile['display_name'] as String?,
      coupleStatus: couple?['status'] as String?,
      members: members,
      pet: pet,
    );
  }
}

class HomePet {
  final Map<String, dynamic> appearance;
  final int version;
  final DateTime updatedAt;

  const HomePet({
    required this.appearance,
    required this.version,
    required this.updatedAt,
  });

  factory HomePet.fromJson(Map<String, dynamic> json) {
    final appearanceData = json['appearance'];
    return HomePet(
      appearance: appearanceData is Map
          ? Map<String, dynamic>.from(appearanceData)
          : const {},
      version: (json['version'] as num).toInt(),
      updatedAt: DateTime.parse(json['updated_at'] as String),
    );
  }
}

class HomeMember {
  final String email;
  final String? nickname;

  const HomeMember({required this.email, required this.nickname});

  factory HomeMember.fromJson(Map<String, dynamic> json) => HomeMember(
    email: json['email'] as String? ?? '',
    nickname: json['nickname'] as String?,
  );
}

class HomeApi {
  final ApiClient _apiClient;

  HomeApi(this._apiClient);

  Future<HomeSnapshot> load() async {
    final response = await _apiClient.dio.get('/home');
    return HomeSnapshot.fromJson(
      Map<String, dynamic>.from(response.data as Map),
    );
  }
}
