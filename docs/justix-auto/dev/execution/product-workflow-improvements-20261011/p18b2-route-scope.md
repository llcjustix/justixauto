# P18b2 route registration seam

P18b1 confirmed that reservation routes register directly in handler/handler.go.
The application hub adds internal/modules/commerce/handler/handler.go to P18b2
ownership solely for the approved continuation proposal/acceptance/conversion
HTTP actions. Domain logic stays in owned service files, with party permission,
exact digest and If-Match checks. No generic credit, custody or expiry endpoint.
This is the same concrete composition seam as P18b1, with no broader refactor
or additional verification gate.
