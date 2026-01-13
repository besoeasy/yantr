# Project Development Guidelines

## APPS Template Standards

Follow the structure and naming conventions outlined in `apps/apps.md` for all application directories and `compose.yml` files. if you need better understanding feel free to pick 5 random apps from the `apps/` directory and study their structure.

## UI Design Standards

### CSS Framework

- **Primary Framework**: Use TailwindCSS for all styling
- **Custom CSS**: Avoid unless absolutely necessary for specific edge cases
- **Rationale**: Maintains consistency and leverages utility-first approach

### User Experience

- **Animations**: Implement smooth transitions and animations throughout the interface
- **Icons**: Use Lucide icons exclusively via `lucide-vue-next` package
- **Design Philosophy**: Follow minimalistic principles prioritizing ease of use
- **Responsiveness**: Ensure all components and pages function properly on mobile devices

---

## API Development Standards

#### Atomicity Requirement (MOST IMPORTANT)

**All Docker API actions MUST be atomic** - they must either:

- Fully succeed with all changes applied, OR
- Fully fail with no partial changes

**Reason**: The application is stateless and does not use a database for tracking changes.

## Post-Edit Checklist

After making any changes to the codebase:

- ✅ Run `node /api/check.js` to verify no conflicts were introduced
- ✅ Test responsive behavior on mobile devices
- ✅ Verify all required labels are present for new apps
