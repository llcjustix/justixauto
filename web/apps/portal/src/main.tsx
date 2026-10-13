import { mountPortal } from '@justixauto/kit';
import './features/realization/design.css';
import './features/financing/design.css';
import './features/admin/design.css';
import { sections } from './sections';

mountPortal({ rootId: 'portal-root', sections });
