# UI Design Guidelines

Ui is located in /ui/

1. Use tailwindcss & aviod custom CSS as much as possible.
2. Use VUEJS framework for all UI components.
3. Follow mobile-first design principles.
4. Use FontAwesome for icons & Avoid using image-based icons.

# Apps

Apps templates located in /apps/, Apps are docker apps which have templates in /apps/<app-name>/compose.yml

1. Avoid using Rootful Ports (e.g., 80, 443). Use high numbered ports (e.g., 8080, 8443). 
2. Use environment variables for configuration wherever possible. 
3. ENsure no ports conflict between different apps. 
4. Use volumes for persistent data storage. never mount to host paths directly. 
5. Follow best practices for security, such as using non-root users and limiting container capabilities.
