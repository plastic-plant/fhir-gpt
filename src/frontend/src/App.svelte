<script>
	import { onMount } from "svelte";
	import Summary from "./Summary.svelte";
	import AllergyIntolerance from "./AllergyIntolerance.svelte";
	import EpisodeOfCare from "./EpisodeOfCare.svelte";

	const components = { Summary, AllergyIntolerance, EpisodeOfCare }; // for bundler
	let component = null;
	let data = null;
	
	const resourceTypes = [
		"Summary",
		"AllergyIntolerance",
		"CarePlan",
		"CareTeam",
		"Composition",
		"Condition",
		"Consent",
		"Coverage",
		"Device",
		"DeviceUseStatement",
		"DiagnosticReport",
		"Encounter",
		"EpisodeOfCare",
		"Flag",
		"Immunization",
		"ImmunizationRecommendation",
		"Location",
		"Media",
		"Medication",
		"NutritionOrder",
		"Observation",
		"Organization",
		"Patient",
		"Practitioner",
		"PractitionerRole",
		"Procedure",
		"RelatedPerson",
		"Specimen",
	];

	onMount(async () => {
		const response = await fetch('http://localhost/fhir/Patient/nl-core-Patient-01?_summary=true&_include=AllergyIntolerance,EpisodeOfCare');
	  	data = await response.json();

		// Can't import with dynamic import(`${resourceType}.svelte`) from url, due to bundler. This will do.
		let resourceType = window.location.hash.slice(1); // i.e. http://localhost:8080/#ResourceType
		setPage(resourceType);
	});

	function setPage(resourceType) {
		// window.location.hash = resourceType;
		switch (resourceType) {
			case "AllergyIntolerance":
				component = AllergyIntolerance;
				break;

			case "EpisodeOfCare":
				component = EpisodeOfCare;
				break;

			default:
				component = Summary;
				break;
		}
	}

	// forgive me, what a mess - should've let the router just do its work ;)
	function remountComponent(event) {
		// event.preventDefault();
		component = null;
		setTimeout(() => {
			setPage(event.target.innerText);
		}, 0);
	}
</script>

<nav>
	<ul>
		{#each resourceTypes as resourceType}
			<li><a href={`#${resourceType}`} on:click={remountComponent}>{resourceType}</a></li>
		{/each}
	</ul>
</nav>

<main>
	{#if component}
		<svelte:component this={component} {data} />
	{/if}
</main>

<style>
	nav {
		float: left;
		margin: 2rem;
		height: 100%;
	}

		ul, li {
			margin: 0;
		}

	main {
		margin: 0 auto;
		margin: 2rem;
	}
</style>
