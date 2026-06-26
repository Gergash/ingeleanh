from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    model_config = SettingsConfigDict(env_file=".env", extra="ignore")

    DATABASE_URL: str = (
        "postgresql+asyncpg://app_user:dev_password_change_me@localhost:5432/proteccion_datos"
    )
    REDIS_URL: str = "redis://:redis_dev_password@localhost:6379/0"
    OPENAI_API_KEY: str = ""
    ENVIRONMENT: str = "development"
    API_PREFIX: str = "/v1"

    CONFIDENCE_HIGH: float = 0.85
    CONFIDENCE_MEDIUM: float = 0.60


settings = Settings()
