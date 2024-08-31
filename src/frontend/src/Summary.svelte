<script>
  export let name = "Summary";
  export let data = "";

  const resources = data.entry.map(entry => entry.resource);
  const summary = resources.find(resource => resource.resourceType === 'Summary');
  const patient = resources.find(resource => resource.resourceType === 'Patient');
</script>

<h4>Summary for patient</h4>
<p class="highlight">{summary?.text ?? "Auto generated summary not found."}</p>

<h4>Patient details</h4>
<!-- <p>{@html patient.text.div}</p> -->
<pre>{JSON.stringify(patient, null, 2)}</pre>

<style>
	.highlight {
		color: blue;
		margin-bottom: 3rem;
	}
</style>