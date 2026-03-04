import './design/index.css';
import './components';
import './elements';
import './views';

import { Router } from '@app/router';

const router = new Router('app-content');
router.start();
