-- Rollback LP-484: Eco 플랜 리소스 단가 원상 복구

UPDATE SYSTEM_SETTINGS
SET setting_value = '30', updated_at = CURRENT_TIMESTAMP
WHERE setting_key = 'eco_cpu_price_per_core_per_minute';

UPDATE SYSTEM_SETTINGS
SET setting_value = '15', updated_at = CURRENT_TIMESTAMP
WHERE setting_key = 'eco_memory_price_per_gb_per_minute';
