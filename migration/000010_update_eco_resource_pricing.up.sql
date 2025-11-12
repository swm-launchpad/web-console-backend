-- LP-484: Eco 플랜 추가 리소스 단가 조정 (93% 인하로 합리적 가격 정책 구현)
-- CPU: ₩30/분 → ₩2.2/분 per core
-- Memory: ₩15/분 → ₩1.1/분 per GB

UPDATE SYSTEM_SETTINGS
SET setting_value = '2.2', updated_at = CURRENT_TIMESTAMP
WHERE setting_key = 'eco_cpu_price_per_core_per_minute';

UPDATE SYSTEM_SETTINGS
SET setting_value = '1.1', updated_at = CURRENT_TIMESTAMP
WHERE setting_key = 'eco_memory_price_per_gb_per_minute';
