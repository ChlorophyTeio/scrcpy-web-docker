# Source provenance

Based on https://github.com/CwithW/ScrcpyCat at `e6f56a7310b3bae6d97570c197cb339317a57925`.

This distribution contains the optional plugin, unified-agent, console and independent MAA worker changes. The inherited licenses and source history are retained. Publication target: https://github.com/ChlorophyTeio/scrcpy-web-docker and Docker Hub namespace `chlorophyimo`. Repository and image addresses are bound in the deployment files and Unraid templates. Release availability is determined by the actual GitHub Release and registry tags.

Local tests cover amd64 runtime images, Chromium WebRTC, Firefox WebSocket fallback, plugin stop/start discovery, MAA native dry-run and arm64 Go compilation. Actual arm64 runtime testing is still required. The publication artifacts were freshly rebuilt from the vendored Android agent and scrcpy server source. The new arm64 Android agent passed a help-only execution probe, and the new scrcpy server passed a display-list probe on the connected Android phone. Other Android ABIs were compiled; no full on-device-agent session or Linux arm64 host runtime was tested in this publication run.
