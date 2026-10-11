import { mountApp } from '@justixauto/kit';
import { navigation } from './navigation';
import './design.css';

mountApp({
  rootId: 'finance-root',
  basename: '/finance',
  brand: 'Исламская линия',
  kinds: ['bank', 'mfo'],
  nav: navigation,
});
