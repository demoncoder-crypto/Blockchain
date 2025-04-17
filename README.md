![Screenshot 2025-04-17 113022](https://github.com/user-attachments/assets/75a7f757-a067-41bf-940d-2ae933f2a342)

    UI <-->|WebSocket| API
    UI -->|REST| DB
    API --> Cache
    DB <-->|reads/writes| Cache
